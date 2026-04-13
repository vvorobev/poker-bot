package hub

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"poker-bot/internal/bot/keyboards"
	"poker-bot/internal/bot/views"
	"poker-bot/internal/domain"
)

const debounceInterval = time.Second

// gameReader is the minimal game-service interface needed by Updater.
type gameReader interface {
	GetGameByID(ctx context.Context, gameID int64) (*domain.Game, error)
	GetParticipants(ctx context.Context, gameID int64) ([]domain.Participant, error)
}

// playerReader is the minimal player-service interface needed by Updater.
type playerReader interface {
	GetPlayer(ctx context.Context, telegramID int64) (*domain.Player, error)
}

// messageEditor is the minimal Telegram interface needed by Updater.
type messageEditor interface {
	EditMessageText(ctx context.Context, params *tgbot.EditMessageTextParams) (*models.Message, error)
}

// Updater rate-limits hub message edits to at most one per second per game.
// Multiple Schedule calls within the debounce window collapse into one editMessageText.
// On Telegram 429 errors the retry_after value from the response is honoured.
type Updater struct {
	editor  messageEditor
	games   gameReader
	players playerReader

	mu      sync.Mutex
	pending map[int64]*time.Timer
}

// NewUpdater creates an Updater. b, games, and players must not be nil.
func NewUpdater(b *tgbot.Bot, games gameReader, players playerReader) *Updater {
	return &Updater{
		editor:  b,
		games:   games,
		players: players,
		pending: make(map[int64]*time.Timer),
	}
}

// newUpdaterWithEditor is used in tests to inject a mock messageEditor.
func newUpdaterWithEditor(editor messageEditor, games gameReader, players playerReader) *Updater {
	return &Updater{
		editor:  editor,
		games:   games,
		players: players,
		pending: make(map[int64]*time.Timer),
	}
}

// Schedule schedules a hub update for gameID. If a pending update already exists
// for this game the debounce timer is reset. ctx is propagated to the eventual
// editMessageText call and any retry triggered by a 429 response.
func (u *Updater) Schedule(ctx context.Context, gameID int64) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if t, ok := u.pending[gameID]; ok {
		t.Reset(debounceInterval)
		return
	}

	u.pending[gameID] = time.AfterFunc(debounceInterval, func() {
		u.mu.Lock()
		delete(u.pending, gameID)
		u.mu.Unlock()
		u.doUpdate(ctx, gameID)
	})
}

// doUpdate reads the current game state from the DB and calls editMessageText.
// On a 429 response it schedules a single retry after retry_after seconds.
func (u *Updater) doUpdate(ctx context.Context, gameID int64) {
	game, err := u.games.GetGameByID(ctx, gameID)
	if err != nil {
		slog.Error("hub updater: GetGameByID", "gameID", gameID, "err", err)
		return
	}
	if game.HubMessageID == 0 {
		return
	}

	participants, err := u.games.GetParticipants(ctx, gameID)
	if err != nil {
		slog.Error("hub updater: GetParticipants", "gameID", gameID, "err", err)
		return
	}

	playerMap := u.buildPlayerMap(ctx, game, participants)
	hubText := views.RenderHub(game, participants, playerMap)

	_, err = u.editor.EditMessageText(ctx, &tgbot.EditMessageTextParams{
		ChatID:      game.ChatID,
		MessageID:   int(game.HubMessageID),
		Text:        hubText,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboards.HubKeyboard(game.ID),
	})
	if err != nil {
		var tooMany *tgbot.TooManyRequestsError
		if errors.As(err, &tooMany) {
			retryAfter := time.Duration(tooMany.RetryAfter) * time.Second
			slog.Warn("hub updater: 429, scheduling retry",
				"gameID", gameID, "retryAfter", retryAfter)
			time.AfterFunc(retryAfter, func() {
				u.doUpdate(ctx, gameID)
			})
			return
		}
		slog.Error("hub updater: EditMessageText", "gameID", gameID, "err", err)
	}
}

func (u *Updater) buildPlayerMap(ctx context.Context, game *domain.Game, participants []domain.Participant) map[int64]*domain.Player {
	ids := make(map[int64]struct{}, len(participants)+1)
	ids[game.CreatorID] = struct{}{}
	for _, p := range participants {
		ids[p.PlayerID] = struct{}{}
	}
	pm := make(map[int64]*domain.Player, len(ids))
	for id := range ids {
		if p, err := u.players.GetPlayer(ctx, id); err == nil {
			pm[id] = p
		}
	}
	return pm
}
