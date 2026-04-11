package service_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"poker-bot/internal/domain"
	"poker-bot/internal/service"
	"poker-bot/internal/storage"
)

func newTestPlayerSvc(t *testing.T) *service.PlayerService {
	t.Helper()
	db := openTestDB(t)
	return service.NewPlayerService(storage.NewPlayerRepo(db))
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	if err := storage.RunMigrations(db); err != nil {
		t.Fatalf("storage.RunMigrations: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestValidatePhone(t *testing.T) {
	valid := []string{"+79991234567", "+70000000000", "+71234567890"}
	for _, p := range valid {
		if !service.ValidatePhone(p) {
			t.Errorf("expected valid: %q", p)
		}
	}

	invalid := []string{"89991234567", "+7abc", "+7999123456", "79991234567", "+8999123456", "", "+7 999 123 45 67"}
	for _, p := range invalid {
		if service.ValidatePhone(p) {
			t.Errorf("expected invalid: %q", p)
		}
	}
}

func TestRegisterAndGetPlayer(t *testing.T) {
	svc := newTestPlayerSvc(t)
	ctx := context.Background()

	err := svc.RegisterPlayer(ctx, 100, "user100", "Alice", "+79991234567", "Тинькофф")
	if err != nil {
		t.Fatalf("RegisterPlayer: %v", err)
	}

	p, err := svc.GetPlayer(ctx, 100)
	if err != nil {
		t.Fatalf("GetPlayer: %v", err)
	}
	if p.DisplayName != "Alice" || p.PhoneNumber != "+79991234567" || p.BankName != "Тинькофф" {
		t.Errorf("unexpected player data: %+v", p)
	}
}

func TestGetPlayer_NotFound(t *testing.T) {
	svc := newTestPlayerSvc(t)
	_, err := svc.GetPlayer(context.Background(), 999)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestIsRegistered(t *testing.T) {
	svc := newTestPlayerSvc(t)
	ctx := context.Background()

	if svc.IsRegistered(ctx, 200) {
		t.Fatal("should not be registered before RegisterPlayer")
	}

	_ = svc.RegisterPlayer(ctx, 200, "u", "Bob", "+79990000000", "Сбербанк")

	if !svc.IsRegistered(ctx, 200) {
		t.Fatal("should be registered after RegisterPlayer")
	}
}

func TestUpdateDisplayName(t *testing.T) {
	svc := newTestPlayerSvc(t)
	ctx := context.Background()

	_ = svc.RegisterPlayer(ctx, 300, "u", "Carol", "+79991111111", "Альфа")

	if err := svc.UpdateDisplayName(ctx, 300, "Карина"); err != nil {
		t.Fatalf("UpdateDisplayName: %v", err)
	}

	p, _ := svc.GetPlayer(ctx, 300)
	if p.DisplayName != "Карина" {
		t.Errorf("expected 'Карина', got %q", p.DisplayName)
	}
}

func TestUpdateDisplayName_NotFound(t *testing.T) {
	svc := newTestPlayerSvc(t)
	err := svc.UpdateDisplayName(context.Background(), 999, "Ghost")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
