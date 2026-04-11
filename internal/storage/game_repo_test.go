package storage

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"poker-bot/internal/domain"
)

// newGameTestDB opens an in-memory DB with migrations applied and returns
// a GameRepo, ParticipantRepo, and a helper to insert a player for FK purposes.
func newGameTestDB(t *testing.T) (*sql.DB, *GameRepo, *ParticipantRepo) {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db, NewGameRepo(db), NewParticipantRepo(db)
}

// insertPlayer inserts a minimal player row to satisfy FK constraints.
func insertPlayer(t *testing.T, db *sql.DB, telegramID int64) {
	t.Helper()
	pr := NewPlayerRepo(db)
	if err := pr.Upsert(context.Background(), &domain.Player{
		TelegramID:  telegramID,
		DisplayName: "TestPlayer",
		PhoneNumber: "+0",
		BankName:    "Bank",
	}); err != nil {
		t.Fatalf("insertPlayer: %v", err)
	}
}

func TestGameRepo_Create_and_GetByID(t *testing.T) {
	db, gr, _ := newGameTestDB(t)
	insertPlayer(t, db, 1)
	ctx := context.Background()

	g := &domain.Game{
		ChatID:    100,
		CreatorID: 1,
		BuyIn:     500,
		Status:    domain.GameStatusActive,
	}
	id, err := gr.Create(ctx, g)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero ID")
	}

	got, err := gr.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ChatID != 100 || got.BuyIn != 500 {
		t.Errorf("unexpected game: %+v", got)
	}
	if got.Status != domain.GameStatusActive {
		t.Errorf("status: want %q, got %q", domain.GameStatusActive, got.Status)
	}
}

func TestGameRepo_GetByID_NotFound(t *testing.T) {
	_, gr, _ := newGameTestDB(t)
	_, err := gr.GetByID(context.Background(), 9999)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGameRepo_GetActiveByChatID(t *testing.T) {
	db, gr, _ := newGameTestDB(t)
	insertPlayer(t, db, 1)
	ctx := context.Background()

	id, _ := gr.Create(ctx, &domain.Game{ChatID: 200, CreatorID: 1, BuyIn: 100, Status: domain.GameStatusActive})

	got, err := gr.GetActiveByChatID(ctx, 200)
	if err != nil {
		t.Fatalf("GetActiveByChatID: %v", err)
	}
	if got.ID != id {
		t.Errorf("want id=%d, got %d", id, got.ID)
	}
}

func TestGameRepo_GetActiveByChatID_NotFound(t *testing.T) {
	_, gr, _ := newGameTestDB(t)
	_, err := gr.GetActiveByChatID(context.Background(), 999)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGameRepo_UpdateStatus(t *testing.T) {
	db, gr, _ := newGameTestDB(t)
	insertPlayer(t, db, 1)
	ctx := context.Background()

	id, _ := gr.Create(ctx, &domain.Game{ChatID: 300, CreatorID: 1, BuyIn: 200, Status: domain.GameStatusActive})

	if err := gr.UpdateStatus(ctx, id, domain.GameStatusFinished); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	got, _ := gr.GetByID(ctx, id)
	if got.Status != domain.GameStatusFinished {
		t.Errorf("want Finished, got %q", got.Status)
	}
}

func TestGameRepo_SetHubMessageID(t *testing.T) {
	db, gr, _ := newGameTestDB(t)
	insertPlayer(t, db, 1)
	ctx := context.Background()

	id, _ := gr.Create(ctx, &domain.Game{ChatID: 400, CreatorID: 1, BuyIn: 100, Status: domain.GameStatusActive})
	if err := gr.SetHubMessageID(ctx, id, 42); err != nil {
		t.Fatalf("SetHubMessageID: %v", err)
	}
	got, _ := gr.GetByID(ctx, id)
	if got.HubMessageID != 42 {
		t.Errorf("want 42, got %d", got.HubMessageID)
	}
}

func TestGameRepo_SetFinishedAt(t *testing.T) {
	db, gr, _ := newGameTestDB(t)
	insertPlayer(t, db, 1)
	ctx := context.Background()

	id, _ := gr.Create(ctx, &domain.Game{ChatID: 500, CreatorID: 1, BuyIn: 100, Status: domain.GameStatusActive})
	ts := time.Now().UTC().Truncate(time.Second)
	if err := gr.SetFinishedAt(ctx, id, ts); err != nil {
		t.Fatalf("SetFinishedAt: %v", err)
	}
	got, _ := gr.GetByID(ctx, id)
	if got.FinishedAt == nil {
		t.Fatal("expected FinishedAt to be set")
	}
}

// ---- ParticipantRepo tests ----

func TestParticipantRepo_Join_and_GetByGameAndPlayer(t *testing.T) {
	db, gr, pr := newGameTestDB(t)
	insertPlayer(t, db, 1)
	ctx := context.Background()

	gameID, _ := gr.Create(ctx, &domain.Game{ChatID: 100, CreatorID: 1, BuyIn: 500, Status: domain.GameStatusActive})

	p := &domain.Participant{GameID: gameID, PlayerID: 1}
	if err := pr.Join(ctx, p); err != nil {
		t.Fatalf("Join: %v", err)
	}

	got, err := pr.GetByGameAndPlayer(ctx, gameID, 1)
	if err != nil {
		t.Fatalf("GetByGameAndPlayer: %v", err)
	}
	if got.GameID != gameID || got.PlayerID != 1 {
		t.Errorf("unexpected participant: %+v", got)
	}
	if got.RebuyCount != 0 {
		t.Errorf("expected rebuy_count=0, got %d", got.RebuyCount)
	}
}

func TestParticipantRepo_Join_ErrAlreadyJoined(t *testing.T) {
	db, gr, pr := newGameTestDB(t)
	insertPlayer(t, db, 1)
	ctx := context.Background()

	gameID, _ := gr.Create(ctx, &domain.Game{ChatID: 100, CreatorID: 1, BuyIn: 500, Status: domain.GameStatusActive})

	_ = pr.Join(ctx, &domain.Participant{GameID: gameID, PlayerID: 1})
	err := pr.Join(ctx, &domain.Participant{GameID: gameID, PlayerID: 1})
	if !errors.Is(err, domain.ErrAlreadyJoined) {
		t.Errorf("expected ErrAlreadyJoined, got %v", err)
	}
}

func TestParticipantRepo_IncrementAndDecrementRebuy(t *testing.T) {
	db, gr, pr := newGameTestDB(t)
	insertPlayer(t, db, 1)
	ctx := context.Background()

	gameID, _ := gr.Create(ctx, &domain.Game{ChatID: 100, CreatorID: 1, BuyIn: 500, Status: domain.GameStatusActive})
	_ = pr.Join(ctx, &domain.Participant{GameID: gameID, PlayerID: 1})

	if err := pr.IncrementRebuy(ctx, gameID, 1); err != nil {
		t.Fatalf("IncrementRebuy: %v", err)
	}
	if err := pr.IncrementRebuy(ctx, gameID, 1); err != nil {
		t.Fatalf("IncrementRebuy 2: %v", err)
	}

	got, _ := pr.GetByGameAndPlayer(ctx, gameID, 1)
	if got.RebuyCount != 2 {
		t.Errorf("expected rebuy_count=2, got %d", got.RebuyCount)
	}

	if err := pr.DecrementRebuy(ctx, gameID, 1); err != nil {
		t.Fatalf("DecrementRebuy: %v", err)
	}
	got, _ = pr.GetByGameAndPlayer(ctx, gameID, 1)
	if got.RebuyCount != 1 {
		t.Errorf("expected rebuy_count=1 after decrement, got %d", got.RebuyCount)
	}
}

func TestParticipantRepo_DecrementRebuy_floor_zero(t *testing.T) {
	db, gr, pr := newGameTestDB(t)
	insertPlayer(t, db, 1)
	ctx := context.Background()

	gameID, _ := gr.Create(ctx, &domain.Game{ChatID: 100, CreatorID: 1, BuyIn: 500, Status: domain.GameStatusActive})
	_ = pr.Join(ctx, &domain.Participant{GameID: gameID, PlayerID: 1})

	// rebuy_count starts at 0; decrement must not go below 0
	if err := pr.DecrementRebuy(ctx, gameID, 1); err != nil {
		t.Fatalf("DecrementRebuy on 0: %v", err)
	}
	got, _ := pr.GetByGameAndPlayer(ctx, gameID, 1)
	if got.RebuyCount != 0 {
		t.Errorf("expected rebuy_count=0 (floor), got %d", got.RebuyCount)
	}
}

func TestParticipantRepo_ListByGame(t *testing.T) {
	db, gr, pr := newGameTestDB(t)
	insertPlayer(t, db, 1)
	insertPlayer(t, db, 2)
	ctx := context.Background()

	gameID, _ := gr.Create(ctx, &domain.Game{ChatID: 100, CreatorID: 1, BuyIn: 500, Status: domain.GameStatusActive})
	_ = pr.Join(ctx, &domain.Participant{GameID: gameID, PlayerID: 1})
	_ = pr.Join(ctx, &domain.Participant{GameID: gameID, PlayerID: 2})

	list, err := pr.ListByGame(ctx, gameID)
	if err != nil {
		t.Fatalf("ListByGame: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 participants, got %d", len(list))
	}
}

func TestParticipantRepo_SetFinalChips(t *testing.T) {
	db, gr, pr := newGameTestDB(t)
	insertPlayer(t, db, 1)
	ctx := context.Background()

	gameID, _ := gr.Create(ctx, &domain.Game{ChatID: 100, CreatorID: 1, BuyIn: 500, Status: domain.GameStatusActive})
	_ = pr.Join(ctx, &domain.Participant{GameID: gameID, PlayerID: 1})

	if err := pr.SetFinalChips(ctx, gameID, 1, 1500); err != nil {
		t.Fatalf("SetFinalChips: %v", err)
	}
	got, _ := pr.GetByGameAndPlayer(ctx, gameID, 1)
	if got.FinalChips == nil || *got.FinalChips != 1500 {
		t.Errorf("expected final_chips=1500, got %v", got.FinalChips)
	}
}

func TestParticipantRepo_SetResultsConfirmed(t *testing.T) {
	db, gr, pr := newGameTestDB(t)
	insertPlayer(t, db, 1)
	ctx := context.Background()

	gameID, _ := gr.Create(ctx, &domain.Game{ChatID: 100, CreatorID: 1, BuyIn: 500, Status: domain.GameStatusActive})
	_ = pr.Join(ctx, &domain.Participant{GameID: gameID, PlayerID: 1})

	if err := pr.SetResultsConfirmed(ctx, gameID, 1); err != nil {
		t.Fatalf("SetResultsConfirmed: %v", err)
	}
	got, _ := pr.GetByGameAndPlayer(ctx, gameID, 1)
	if !got.ResultsConfirmed {
		t.Error("expected results_confirmed=true")
	}
}
