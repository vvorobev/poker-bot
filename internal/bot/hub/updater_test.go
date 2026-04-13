package hub

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"poker-bot/internal/domain"
)

// --- fakes ---

type fakeGame struct {
	game         *domain.Game
	participants []domain.Participant
}

func (f *fakeGame) GetGameByID(_ context.Context, _ int64) (*domain.Game, error) {
	return f.game, nil
}
func (f *fakeGame) GetParticipants(_ context.Context, _ int64) ([]domain.Participant, error) {
	return f.participants, nil
}

type fakePlayers struct{}

func (f *fakePlayers) GetPlayer(_ context.Context, id int64) (*domain.Player, error) {
	return &domain.Player{TelegramID: id, DisplayName: "Player"}, nil
}

type fakeEditor struct {
	mu    sync.Mutex
	calls int32
	err   error
}

func (f *fakeEditor) EditMessageText(_ context.Context, _ *tgbot.EditMessageTextParams) (*models.Message, error) {
	atomic.AddInt32(&f.calls, 1)
	f.mu.Lock()
	err := f.err
	f.mu.Unlock()
	return nil, err
}

func (f *fakeEditor) callCount() int {
	return int(atomic.LoadInt32(&f.calls))
}

func newTestUpdater(editor *fakeEditor) *Updater {
	game := &domain.Game{
		ID:           42,
		ChatID:       100,
		CreatorID:    1,
		BuyIn:        500,
		HubMessageID: 7,
		Status:       domain.GameStatusActive,
	}
	return newUpdaterWithEditor(editor, &fakeGame{game: game}, &fakePlayers{})
}

// TestDebounce: 5 Schedule calls in 100ms → exactly 1 editMessageText.
func TestDebounce(t *testing.T) {
	editor := &fakeEditor{}
	u := newTestUpdater(editor)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		u.Schedule(ctx, 42)
		time.Sleep(20 * time.Millisecond)
	}

	// Wait for debounce to fire (debounceInterval = 1s from last Schedule).
	time.Sleep(debounceInterval + 200*time.Millisecond)

	if got := editor.callCount(); got != 1 {
		t.Errorf("want 1 editMessageText call, got %d", got)
	}
}

// TestDebounceIndependentGames: Schedule for two different games → 2 edits.
func TestDebounceIndependentGames(t *testing.T) {
	editor := &fakeEditor{}
	game1 := &domain.Game{ID: 1, ChatID: 10, CreatorID: 1, BuyIn: 100, HubMessageID: 1, Status: domain.GameStatusActive}
	game2 := &domain.Game{ID: 2, ChatID: 20, CreatorID: 1, BuyIn: 100, HubMessageID: 2, Status: domain.GameStatusActive}

	u := &Updater{
		editor:  editor,
		pending: make(map[int64]*time.Timer),
		games: &dualGameReader{
			games: map[int64]*domain.Game{1: game1, 2: game2},
		},
		players: &fakePlayers{},
	}

	ctx := context.Background()
	u.Schedule(ctx, 1)
	u.Schedule(ctx, 2)

	time.Sleep(debounceInterval + 200*time.Millisecond)

	if got := editor.callCount(); got != 2 {
		t.Errorf("want 2 editMessageText calls, got %d", got)
	}
}

// TestRateLimitRetry: 429 response causes a single retry after retry_after.
func TestRateLimitRetry(t *testing.T) {
	editor := &fakeEditor{}
	u := newTestUpdater(editor)
	ctx := context.Background()

	// First call returns 429 with retry_after=1.
	tooMany := &tgbot.TooManyRequestsError{Message: "Too Many Requests", RetryAfter: 1}
	editor.mu.Lock()
	editor.err = tooMany
	editor.mu.Unlock()

	u.doUpdate(ctx, 42)

	// After 429, 0 successful calls yet (both calls errored).
	time.Sleep(50 * time.Millisecond)
	if got := editor.callCount(); got != 1 {
		t.Errorf("want 1 call right after 429, got %d", got)
	}

	// Clear the error before retry fires.
	editor.mu.Lock()
	editor.err = nil
	editor.mu.Unlock()

	// Retry fires after 1s.
	time.Sleep(1200 * time.Millisecond)
	if got := editor.callCount(); got != 2 {
		t.Errorf("want 2 total calls after retry, got %d", got)
	}
}

// TestNoUpdateWhenHubMessageIDZero: game with HubMessageID=0 → no edit.
func TestNoUpdateWhenHubMessageIDZero(t *testing.T) {
	editor := &fakeEditor{}
	game := &domain.Game{ID: 99, ChatID: 10, CreatorID: 1, BuyIn: 100, HubMessageID: 0, Status: domain.GameStatusActive}
	u := newUpdaterWithEditor(editor, &fakeGame{game: game}, &fakePlayers{})

	u.doUpdate(context.Background(), 99)
	time.Sleep(50 * time.Millisecond)

	if got := editor.callCount(); got != 0 {
		t.Errorf("want 0 calls for HubMessageID=0, got %d", got)
	}
}

// TestNonTelegramError: non-429 error is logged but no retry.
func TestNonTelegramError(t *testing.T) {
	editor := &fakeEditor{err: errors.New("internal server error")}
	u := newTestUpdater(editor)

	u.doUpdate(context.Background(), 42)
	time.Sleep(50 * time.Millisecond)

	if got := editor.callCount(); got != 1 {
		t.Errorf("want exactly 1 call (no retry), got %d", got)
	}
}

// --- helpers ---

type dualGameReader struct {
	games map[int64]*domain.Game
}

func (d *dualGameReader) GetGameByID(_ context.Context, id int64) (*domain.Game, error) {
	if g, ok := d.games[id]; ok {
		return g, nil
	}
	return nil, errors.New("not found")
}

func (d *dualGameReader) GetParticipants(_ context.Context, _ int64) ([]domain.Participant, error) {
	return nil, nil
}
