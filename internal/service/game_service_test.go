package service_test

import (
	"context"
	"errors"
	"testing"

	"poker-bot/internal/domain"
	"poker-bot/internal/service"
	"poker-bot/internal/storage"
)

func newTestGameSvc(t *testing.T) *service.GameService {
	t.Helper()
	db := openTestDB(t)
	games := storage.NewGameRepo(db)
	participants := storage.NewParticipantRepo(db)
	tx := storage.NewTxManager(db)
	return service.NewGameService(games, participants, tx)
}

// registerPlayer is a helper to register a player in the test DB via PlayerService.
func registerPlayerForGame(t *testing.T, svc *service.GameService, db interface{}) {
	t.Helper()
}

// ─── TASK-019 tests ───────────────────────────────────────────────────────────

func TestNewGame_Success(t *testing.T) {
	db := openTestDB(t)
	games := storage.NewGameRepo(db)
	participants := storage.NewParticipantRepo(db)
	tx := storage.NewTxManager(db)
	playerRepo := storage.NewPlayerRepo(db)
	svc := service.NewGameService(games, participants, tx)
	ctx := context.Background()

	// Register creator
	creator := testPlayer(1001)
	if err := playerRepo.Upsert(ctx, creator); err != nil {
		t.Fatalf("Upsert creator: %v", err)
	}

	game, err := svc.NewGame(ctx, 42, 1001, 1000)
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	if game.ID == 0 {
		t.Fatal("expected non-zero game ID")
	}
	if game.Status != domain.GameStatusActive {
		t.Errorf("expected status active, got %q", game.Status)
	}
	if game.BuyIn != 1000 {
		t.Errorf("expected buyIn=1000, got %d", game.BuyIn)
	}

	// Creator must be in participants
	pp, err := participants.ListByGame(ctx, game.ID)
	if err != nil {
		t.Fatalf("ListByGame: %v", err)
	}
	if len(pp) != 1 || pp[0].PlayerID != 1001 {
		t.Errorf("expected creator in participants, got %+v", pp)
	}
}

func TestNewGame_ErrGameAlreadyActive(t *testing.T) {
	db := openTestDB(t)
	games := storage.NewGameRepo(db)
	participants := storage.NewParticipantRepo(db)
	tx := storage.NewTxManager(db)
	playerRepo := storage.NewPlayerRepo(db)
	svc := service.NewGameService(games, participants, tx)
	ctx := context.Background()

	if err := playerRepo.Upsert(ctx, testPlayer(1002)); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	if _, err := svc.NewGame(ctx, 55, 1002, 1000); err != nil {
		t.Fatalf("first NewGame: %v", err)
	}

	_, err := svc.NewGame(ctx, 55, 1002, 1000)
	if !errors.Is(err, domain.ErrGameAlreadyActive) {
		t.Fatalf("expected ErrGameAlreadyActive, got %v", err)
	}
}

func TestNewGame_BuyInValidation(t *testing.T) {
	db := openTestDB(t)
	games := storage.NewGameRepo(db)
	participants := storage.NewParticipantRepo(db)
	tx := storage.NewTxManager(db)
	playerRepo := storage.NewPlayerRepo(db)
	svc := service.NewGameService(games, participants, tx)
	ctx := context.Background()

	if err := playerRepo.Upsert(ctx, testPlayer(1003)); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	if _, err := svc.NewGame(ctx, 66, 1003, 50); err == nil {
		t.Fatal("expected error for buyIn=50")
	}
	if _, err := svc.NewGame(ctx, 66, 1003, 200_000); err == nil {
		t.Fatal("expected error for buyIn=200000")
	}
	// Edge values must succeed
	if _, err := svc.NewGame(ctx, 66, 1003, 100); err != nil {
		t.Fatalf("buyIn=100 should succeed: %v", err)
	}
}

func TestGetActiveGame(t *testing.T) {
	db := openTestDB(t)
	games := storage.NewGameRepo(db)
	participants := storage.NewParticipantRepo(db)
	tx := storage.NewTxManager(db)
	playerRepo := storage.NewPlayerRepo(db)
	svc := service.NewGameService(games, participants, tx)
	ctx := context.Background()

	// No game yet
	_, err := svc.GetActiveGame(ctx, 77)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound before any game, got %v", err)
	}

	if err := playerRepo.Upsert(ctx, testPlayer(1004)); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	created, _ := svc.NewGame(ctx, 77, 1004, 500)
	got, err := svc.GetActiveGame(ctx, 77)
	if err != nil {
		t.Fatalf("GetActiveGame: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("expected game ID %d, got %d", created.ID, got.ID)
	}
}

// ─── TASK-024 tests ───────────────────────────────────────────────────────────

func TestJoin_Success(t *testing.T) {
	db := openTestDB(t)
	games := storage.NewGameRepo(db)
	parts := storage.NewParticipantRepo(db)
	tx := storage.NewTxManager(db)
	playerRepo := storage.NewPlayerRepo(db)
	svc := service.NewGameService(games, parts, tx)
	ctx := context.Background()

	if err := playerRepo.Upsert(ctx, testPlayer(2001)); err != nil {
		t.Fatalf("Upsert creator: %v", err)
	}
	if err := playerRepo.Upsert(ctx, testPlayer(2002)); err != nil {
		t.Fatalf("Upsert joiner: %v", err)
	}

	game, _ := svc.NewGame(ctx, 100, 2001, 1000)

	_, pp, err := svc.Join(ctx, game.ID, 2002)
	if err != nil {
		t.Fatalf("Join: %v", err)
	}
	if len(pp) != 2 {
		t.Errorf("expected 2 participants, got %d", len(pp))
	}
}

func TestJoin_ErrAlreadyJoined(t *testing.T) {
	db := openTestDB(t)
	games := storage.NewGameRepo(db)
	parts := storage.NewParticipantRepo(db)
	tx := storage.NewTxManager(db)
	playerRepo := storage.NewPlayerRepo(db)
	svc := service.NewGameService(games, parts, tx)
	ctx := context.Background()

	if err := playerRepo.Upsert(ctx, testPlayer(3001)); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	game, _ := svc.NewGame(ctx, 200, 3001, 1000)

	// 3001 is already a participant (added by NewGame)
	_, _, err := svc.Join(ctx, game.ID, 3001)
	if !errors.Is(err, domain.ErrAlreadyJoined) {
		t.Fatalf("expected ErrAlreadyJoined, got %v", err)
	}
}

func TestRebuyAndCancelRebuy(t *testing.T) {
	db := openTestDB(t)
	games := storage.NewGameRepo(db)
	parts := storage.NewParticipantRepo(db)
	tx := storage.NewTxManager(db)
	playerRepo := storage.NewPlayerRepo(db)
	svc := service.NewGameService(games, parts, tx)
	ctx := context.Background()

	if err := playerRepo.Upsert(ctx, testPlayer(4001)); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	game, _ := svc.NewGame(ctx, 300, 4001, 1000)

	// Rebuy 3 times
	for i := 0; i < 3; i++ {
		if _, _, err := svc.Rebuy(ctx, game.ID, 4001); err != nil {
			t.Fatalf("Rebuy %d: %v", i+1, err)
		}
	}

	// CancelRebuy 1 time → rebuy_count should be 2
	_, pp, err := svc.CancelRebuy(ctx, game.ID, 4001)
	if err != nil {
		t.Fatalf("CancelRebuy: %v", err)
	}
	if len(pp) == 0 || pp[0].RebuyCount != 2 {
		t.Errorf("expected rebuy_count=2, got %+v", pp)
	}
}

func TestCancelRebuy_FloorAtZero(t *testing.T) {
	db := openTestDB(t)
	games := storage.NewGameRepo(db)
	parts := storage.NewParticipantRepo(db)
	tx := storage.NewTxManager(db)
	playerRepo := storage.NewPlayerRepo(db)
	svc := service.NewGameService(games, parts, tx)
	ctx := context.Background()

	if err := playerRepo.Upsert(ctx, testPlayer(5001)); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	game, _ := svc.NewGame(ctx, 400, 5001, 1000)

	// CancelRebuy at zero — should not error, count stays 0
	_, pp, err := svc.CancelRebuy(ctx, game.ID, 5001)
	if err != nil {
		t.Fatalf("CancelRebuy at zero: %v", err)
	}
	if len(pp) == 0 || pp[0].RebuyCount != 0 {
		t.Errorf("expected rebuy_count=0, got %+v", pp)
	}
}

func TestRebuy_ErrNotParticipant(t *testing.T) {
	db := openTestDB(t)
	games := storage.NewGameRepo(db)
	parts := storage.NewParticipantRepo(db)
	tx := storage.NewTxManager(db)
	playerRepo := storage.NewPlayerRepo(db)
	svc := service.NewGameService(games, parts, tx)
	ctx := context.Background()

	if err := playerRepo.Upsert(ctx, testPlayer(6001)); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	game, _ := svc.NewGame(ctx, 500, 6001, 1000)

	_, _, err := svc.Rebuy(ctx, game.ID, 9999)
	if !errors.Is(err, domain.ErrNotParticipant) {
		t.Fatalf("expected ErrNotParticipant, got %v", err)
	}
}

// ─── TASK-028 tests ───────────────────────────────────────────────────────────

func TestFinishGame_Success(t *testing.T) {
	db := openTestDB(t)
	games := storage.NewGameRepo(db)
	parts := storage.NewParticipantRepo(db)
	tx := storage.NewTxManager(db)
	playerRepo := storage.NewPlayerRepo(db)
	svc := service.NewGameService(games, parts, tx)
	ctx := context.Background()

	if err := playerRepo.Upsert(ctx, testPlayer(7001)); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := playerRepo.Upsert(ctx, testPlayer(7002)); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := playerRepo.Upsert(ctx, testPlayer(7003)); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	game, _ := svc.NewGame(ctx, 600, 7001, 1000)
	svc.Join(ctx, game.ID, 7002) //nolint
	svc.Join(ctx, game.ID, 7003) //nolint

	g, pp, err := svc.FinishGame(ctx, game.ID, 7001)
	if err != nil {
		t.Fatalf("FinishGame: %v", err)
	}
	if g.Status != domain.GameStatusCollectingResults {
		t.Errorf("expected CollectingResults, got %q", g.Status)
	}
	if len(pp) != 3 {
		t.Errorf("expected 3 participants, got %d", len(pp))
	}
}

func TestFinishGame_ErrGameNotActive(t *testing.T) {
	db := openTestDB(t)
	games := storage.NewGameRepo(db)
	parts := storage.NewParticipantRepo(db)
	tx := storage.NewTxManager(db)
	playerRepo := storage.NewPlayerRepo(db)
	svc := service.NewGameService(games, parts, tx)
	ctx := context.Background()

	if err := playerRepo.Upsert(ctx, testPlayer(8001)); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	game, _ := svc.NewGame(ctx, 700, 8001, 1000)
	svc.FinishGame(ctx, game.ID, 8001) //nolint: transition to CollectingResults

	// Second call must fail
	_, _, err := svc.FinishGame(ctx, game.ID, 8001)
	if !errors.Is(err, domain.ErrGameNotActive) {
		t.Fatalf("expected ErrGameNotActive, got %v", err)
	}
}

func TestFinishGame_ErrNotParticipant(t *testing.T) {
	db := openTestDB(t)
	games := storage.NewGameRepo(db)
	parts := storage.NewParticipantRepo(db)
	tx := storage.NewTxManager(db)
	playerRepo := storage.NewPlayerRepo(db)
	svc := service.NewGameService(games, parts, tx)
	ctx := context.Background()

	if err := playerRepo.Upsert(ctx, testPlayer(9001)); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	game, _ := svc.NewGame(ctx, 800, 9001, 1000)

	_, _, err := svc.FinishGame(ctx, game.ID, 9999)
	if !errors.Is(err, domain.ErrNotParticipant) {
		t.Fatalf("expected ErrNotParticipant, got %v", err)
	}
}

// testPlayer returns a minimal domain.Player for test setup.
func testPlayer(id int64) *domain.Player {
	return &domain.Player{
		TelegramID:       id,
		TelegramUsername: "user",
		DisplayName:      "Test User",
		PhoneNumber:      "+79991234567",
		BankName:         "TestBank",
	}
}
