package storage

import (
	"context"
	"errors"
	"testing"

	"poker-bot/internal/domain"
)

func newTestDB(t *testing.T) *PlayerRepo {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewPlayerRepo(db)
}

func TestPlayerRepo_Upsert_and_GetByTelegramID(t *testing.T) {
	repo := newTestDB(t)
	ctx := context.Background()

	p := &domain.Player{
		TelegramID:       12345,
		TelegramUsername: "alice",
		DisplayName:      "Alice",
		PhoneNumber:      "+1234567890",
		BankName:         "Sberbank",
	}

	if err := repo.Upsert(ctx, p); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := repo.GetByTelegramID(ctx, 12345)
	if err != nil {
		t.Fatalf("GetByTelegramID: %v", err)
	}
	if got.TelegramUsername != "alice" {
		t.Errorf("username: want %q, got %q", "alice", got.TelegramUsername)
	}
	if got.BankName != "Sberbank" {
		t.Errorf("bank_name: want %q, got %q", "Sberbank", got.BankName)
	}
}

func TestPlayerRepo_GetByTelegramID_NotFound(t *testing.T) {
	repo := newTestDB(t)
	ctx := context.Background()

	_, err := repo.GetByTelegramID(ctx, 99999)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPlayerRepo_Upsert_updates_existing(t *testing.T) {
	repo := newTestDB(t)
	ctx := context.Background()

	p := &domain.Player{
		TelegramID:       42,
		TelegramUsername: "bob",
		DisplayName:      "Bob",
		PhoneNumber:      "+7000",
		BankName:         "Tinkoff",
	}
	if err := repo.Upsert(ctx, p); err != nil {
		t.Fatalf("first Upsert: %v", err)
	}

	p.BankName = "VTB"
	if err := repo.Upsert(ctx, p); err != nil {
		t.Fatalf("second Upsert: %v", err)
	}

	got, err := repo.GetByTelegramID(ctx, 42)
	if err != nil {
		t.Fatalf("GetByTelegramID: %v", err)
	}
	if got.BankName != "VTB" {
		t.Errorf("expected bank_name=VTB after update, got %q", got.BankName)
	}
}

func TestPlayerRepo_RunInTx_commit(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()
	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	repo := NewPlayerRepo(db)
	txm := NewTxManager(db)
	ctx := context.Background()

	err = txm.RunInTx(ctx, func(txCtx context.Context) error {
		return repo.Upsert(txCtx, &domain.Player{
			TelegramID:  100,
			DisplayName: "TxPlayer",
			PhoneNumber: "+0",
			BankName:    "BankA",
		})
	})
	if err != nil {
		t.Fatalf("RunInTx: %v", err)
	}

	got, err := repo.GetByTelegramID(ctx, 100)
	if err != nil {
		t.Fatalf("GetByTelegramID after commit: %v", err)
	}
	if got.DisplayName != "TxPlayer" {
		t.Errorf("expected TxPlayer, got %q", got.DisplayName)
	}
}

func TestPlayerRepo_RunInTx_rollback(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()
	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	repo := NewPlayerRepo(db)
	txm := NewTxManager(db)
	ctx := context.Background()

	sentinelErr := errors.New("abort")
	err = txm.RunInTx(ctx, func(txCtx context.Context) error {
		_ = repo.Upsert(txCtx, &domain.Player{
			TelegramID:  200,
			DisplayName: "RollbackPlayer",
			PhoneNumber: "+0",
			BankName:    "BankB",
		})
		return sentinelErr
	})
	if !errors.Is(err, sentinelErr) {
		t.Fatalf("expected sentinelErr, got %v", err)
	}

	_, err = repo.GetByTelegramID(ctx, 200)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound after rollback, got %v", err)
	}
}
