package storage

import (
	"context"
	"database/sql"
	"fmt"

	"poker-bot/internal/domain"
)

// SettlementRepo implements service.SettlementRepository against SQLite.
type SettlementRepo struct {
	db *sql.DB
}

// NewSettlementRepo creates a SettlementRepo backed by db.
func NewSettlementRepo(db *sql.DB) *SettlementRepo {
	return &SettlementRepo{db: db}
}

// SaveAll inserts all transfers for a game in a single operation.
// If transfers is empty, it is a no-op.
func (r *SettlementRepo) SaveAll(ctx context.Context, gameID int64, transfers []domain.Transfer) error {
	if len(transfers) == 0 {
		return nil
	}

	q := extractDB(ctx, r.db)
	for _, t := range transfers {
		_, err := q.ExecContext(ctx,
			`INSERT INTO settlements (game_id, from_player_id, to_player_id, amount) VALUES (?, ?, ?, ?)`,
			gameID, t.FromPlayerID, t.ToPlayerID, t.Amount,
		)
		if err != nil {
			return fmt.Errorf("storage.SettlementRepo.SaveAll: %w", err)
		}
	}
	return nil
}

// ListByGame returns all settlements for a game.
func (r *SettlementRepo) ListByGame(ctx context.Context, gameID int64) ([]domain.Settlement, error) {
	q := extractDB(ctx, r.db)
	rows, err := q.QueryContext(ctx,
		`SELECT id, game_id, from_player_id, to_player_id, amount FROM settlements WHERE game_id = ?`,
		gameID,
	)
	if err != nil {
		return nil, fmt.Errorf("storage.SettlementRepo.ListByGame: %w", err)
	}
	defer rows.Close()

	var settlements []domain.Settlement
	for rows.Next() {
		var s domain.Settlement
		if err := rows.Scan(&s.ID, &s.GameID, &s.FromPlayerID, &s.ToPlayerID, &s.Amount); err != nil {
			return nil, fmt.Errorf("storage.SettlementRepo.ListByGame scan: %w", err)
		}
		settlements = append(settlements, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage.SettlementRepo.ListByGame rows: %w", err)
	}
	return settlements, nil
}
