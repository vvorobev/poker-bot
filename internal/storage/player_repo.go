package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"poker-bot/internal/domain"
)

// PlayerRepo implements service.PlayerRepository against SQLite.
type PlayerRepo struct {
	db *sql.DB
}

// NewPlayerRepo creates a PlayerRepo backed by db.
func NewPlayerRepo(db *sql.DB) *PlayerRepo {
	return &PlayerRepo{db: db}
}

// GetByTelegramID fetches a player by telegram_id.
// Returns domain.ErrNotFound when no row exists.
func (r *PlayerRepo) GetByTelegramID(ctx context.Context, telegramID int64) (*domain.Player, error) {
	q := extractDB(ctx, r.db)
	row := q.QueryRowContext(ctx, `
		SELECT telegram_id, telegram_username, display_name, phone_number, bank_name, created_at, updated_at
		FROM players
		WHERE telegram_id = ?`, telegramID)

	p := &domain.Player{}
	err := row.Scan(
		&p.TelegramID,
		&p.TelegramUsername,
		&p.DisplayName,
		&p.PhoneNumber,
		&p.BankName,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("storage.PlayerRepo.GetByTelegramID: %w", err)
	}
	return p, nil
}

// Upsert inserts or updates a player record.
// updated_at is always set to the current UTC time.
func (r *PlayerRepo) Upsert(ctx context.Context, p *domain.Player) error {
	q := extractDB(ctx, r.db)
	now := time.Now().UTC()

	_, err := q.ExecContext(ctx, `
		INSERT INTO players (telegram_id, telegram_username, display_name, phone_number, bank_name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(telegram_id) DO UPDATE SET
			telegram_username = excluded.telegram_username,
			display_name      = excluded.display_name,
			phone_number      = excluded.phone_number,
			bank_name         = excluded.bank_name,
			updated_at        = excluded.updated_at`,
		p.TelegramID,
		p.TelegramUsername,
		p.DisplayName,
		p.PhoneNumber,
		p.BankName,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("storage.PlayerRepo.Upsert: %w", err)
	}
	return nil
}
