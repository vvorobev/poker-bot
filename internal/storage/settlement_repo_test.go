package storage

import (
	"context"
	"errors"
	"testing"

	"poker-bot/internal/domain"
)

func newSettlementTestDB(t *testing.T) (*SettlementRepo, func(fromID, toID int64) int64) {
	t.Helper()
	db, gr, _ := newGameTestDB(t)
	sr := NewSettlementRepo(db)

	makeGame := func(fromID, toID int64) int64 {
		insertPlayer(t, db, fromID)
		insertPlayer(t, db, toID)
		ctx := context.Background()
		id, err := gr.Create(ctx, &domain.Game{
			ChatID:    100,
			CreatorID: fromID,
			BuyIn:     500,
			Status:    domain.GameStatusActive,
		})
		if err != nil {
			t.Fatalf("Create game: %v", err)
		}
		return id
	}
	return sr, makeGame
}

func TestSettlementRepo_SaveAll_and_ListByGame(t *testing.T) {
	sr, makeGame := newSettlementTestDB(t)
	gameID := makeGame(1, 2)
	ctx := context.Background()

	transfers := []domain.Transfer{
		{FromPlayerID: 1, ToPlayerID: 2, Amount: 1000},
		{FromPlayerID: 2, ToPlayerID: 1, Amount: 500},
	}
	if err := sr.SaveAll(ctx, gameID, transfers); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}

	got, err := sr.ListByGame(ctx, gameID)
	if err != nil {
		t.Fatalf("ListByGame: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 settlements, got %d", len(got))
	}
	if got[0].FromPlayerID != 1 || got[0].ToPlayerID != 2 || got[0].Amount != 1000 {
		t.Errorf("settlement[0] mismatch: %+v", got[0])
	}
	if got[1].FromPlayerID != 2 || got[1].ToPlayerID != 1 || got[1].Amount != 500 {
		t.Errorf("settlement[1] mismatch: %+v", got[1])
	}
}

func TestSettlementRepo_SaveAll_empty(t *testing.T) {
	sr, makeGame := newSettlementTestDB(t)
	gameID := makeGame(10, 11)
	ctx := context.Background()

	if err := sr.SaveAll(ctx, gameID, nil); err != nil {
		t.Fatalf("SaveAll empty: %v", err)
	}
	got, err := sr.ListByGame(ctx, gameID)
	if err != nil {
		t.Fatalf("ListByGame: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 settlements, got %d", len(got))
	}
}

func TestSettlementRepo_SaveAll_in_transaction(t *testing.T) {
	db, gr, _ := newGameTestDB(t)
	sr := NewSettlementRepo(db)
	txm := NewTxManager(db)
	ctx := context.Background()

	insertPlayer(t, db, 20)
	insertPlayer(t, db, 21)
	gameID, err := gr.Create(ctx, &domain.Game{
		ChatID:    200,
		CreatorID: 20,
		BuyIn:     1000,
		Status:    domain.GameStatusActive,
	})
	if err != nil {
		t.Fatalf("Create game: %v", err)
	}

	sentinelErr := errors.New("rollback")
	_ = txm.RunInTx(ctx, func(txCtx context.Context) error {
		_ = sr.SaveAll(txCtx, gameID, []domain.Transfer{
			{FromPlayerID: 20, ToPlayerID: 21, Amount: 300},
		})
		return sentinelErr
	})

	got, err := sr.ListByGame(ctx, gameID)
	if err != nil {
		t.Fatalf("ListByGame: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 settlements after rollback, got %d", len(got))
	}
}

func TestSettlementRepo_ListByGame_empty(t *testing.T) {
	sr, makeGame := newSettlementTestDB(t)
	gameID := makeGame(30, 31)
	ctx := context.Background()

	got, err := sr.ListByGame(ctx, gameID)
	if err != nil {
		t.Fatalf("ListByGame: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}
