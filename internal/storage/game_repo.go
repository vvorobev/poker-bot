package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"poker-bot/internal/domain"
)

// GameRepo implements service.GameRepository against SQLite.
type GameRepo struct {
	db *sql.DB
}

// NewGameRepo creates a GameRepo backed by db.
func NewGameRepo(db *sql.DB) *GameRepo {
	return &GameRepo{db: db}
}

// Create inserts a new game and returns its generated ID.
func (r *GameRepo) Create(ctx context.Context, g *domain.Game) (int64, error) {
	q := extractDB(ctx, r.db)
	res, err := q.ExecContext(ctx, `
		INSERT INTO games (chat_id, creator_id, buy_in, hub_message_id, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		g.ChatID,
		g.CreatorID,
		g.BuyIn,
		g.HubMessageID,
		g.Status,
		time.Now().UTC(),
	)
	if err != nil {
		return 0, fmt.Errorf("storage.GameRepo.Create: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("storage.GameRepo.Create LastInsertId: %w", err)
	}
	return id, nil
}

// GetByID fetches a game by primary key. Returns domain.ErrNotFound if absent.
func (r *GameRepo) GetByID(ctx context.Context, id int64) (*domain.Game, error) {
	q := extractDB(ctx, r.db)
	row := q.QueryRowContext(ctx, `
		SELECT id, chat_id, creator_id, buy_in, hub_message_id, status, created_at, finished_at
		FROM games WHERE id = ?`, id)
	return scanGame(row)
}

// GetActiveByChatID returns the single active game for a chat.
// Returns domain.ErrNotFound when no active game exists.
func (r *GameRepo) GetActiveByChatID(ctx context.Context, chatID int64) (*domain.Game, error) {
	q := extractDB(ctx, r.db)
	row := q.QueryRowContext(ctx, `
		SELECT id, chat_id, creator_id, buy_in, hub_message_id, status, created_at, finished_at
		FROM games WHERE chat_id = ? AND status = ?`, chatID, domain.GameStatusActive)
	return scanGame(row)
}

// GetCollectingByPlayerID returns the game in collecting_results status where playerID is a participant.
// Returns domain.ErrNotFound when no such game exists.
func (r *GameRepo) GetCollectingByPlayerID(ctx context.Context, playerID int64) (*domain.Game, error) {
	q := extractDB(ctx, r.db)
	row := q.QueryRowContext(ctx, `
		SELECT g.id, g.chat_id, g.creator_id, g.buy_in, g.hub_message_id, g.status, g.created_at, g.finished_at
		FROM games g
		JOIN game_participants p ON p.game_id = g.id
		WHERE p.player_id = ? AND g.status = ?
		LIMIT 1`, playerID, domain.GameStatusCollectingResults)
	return scanGame(row)
}

// GetFinishedByPlayerID returns the most recent finished game where playerID is a participant.
// Returns domain.ErrNotFound when no such game exists.
func (r *GameRepo) GetFinishedByPlayerID(ctx context.Context, playerID int64) (*domain.Game, error) {
	q := extractDB(ctx, r.db)
	row := q.QueryRowContext(ctx, `
		SELECT g.id, g.chat_id, g.creator_id, g.buy_in, g.hub_message_id, g.status, g.created_at, g.finished_at
		FROM games g
		JOIN game_participants p ON p.game_id = g.id
		WHERE p.player_id = ? AND g.status = ?
		ORDER BY g.finished_at DESC
		LIMIT 1`, playerID, domain.GameStatusFinished)
	return scanGame(row)
}

// UpdateStatus sets the status of a game.
func (r *GameRepo) UpdateStatus(ctx context.Context, id int64, status domain.GameStatus) error {
	q := extractDB(ctx, r.db)
	_, err := q.ExecContext(ctx, `UPDATE games SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return fmt.Errorf("storage.GameRepo.UpdateStatus: %w", err)
	}
	return nil
}

// SetHubMessageID records the Telegram message ID for the hub message.
func (r *GameRepo) SetHubMessageID(ctx context.Context, id int64, msgID int64) error {
	q := extractDB(ctx, r.db)
	_, err := q.ExecContext(ctx, `UPDATE games SET hub_message_id = ? WHERE id = ?`, msgID, id)
	if err != nil {
		return fmt.Errorf("storage.GameRepo.SetHubMessageID: %w", err)
	}
	return nil
}

// SetFinishedAt marks a game as finished at the given time.
func (r *GameRepo) SetFinishedAt(ctx context.Context, id int64, t time.Time) error {
	q := extractDB(ctx, r.db)
	_, err := q.ExecContext(ctx, `UPDATE games SET finished_at = ? WHERE id = ?`, t.UTC(), id)
	if err != nil {
		return fmt.Errorf("storage.GameRepo.SetFinishedAt: %w", err)
	}
	return nil
}

func scanGame(row *sql.Row) (*domain.Game, error) {
	g := &domain.Game{}
	err := row.Scan(
		&g.ID,
		&g.ChatID,
		&g.CreatorID,
		&g.BuyIn,
		&g.HubMessageID,
		&g.Status,
		&g.CreatedAt,
		&g.FinishedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("storage.GameRepo scan: %w", err)
	}
	return g, nil
}

// ParticipantRepo implements service.ParticipantRepository against SQLite.
type ParticipantRepo struct {
	db *sql.DB
}

// NewParticipantRepo creates a ParticipantRepo backed by db.
func NewParticipantRepo(db *sql.DB) *ParticipantRepo {
	return &ParticipantRepo{db: db}
}

// Join inserts a new participant. Returns domain.ErrAlreadyJoined on UNIQUE conflict.
func (r *ParticipantRepo) Join(ctx context.Context, p *domain.Participant) error {
	q := extractDB(ctx, r.db)
	_, err := q.ExecContext(ctx, `
		INSERT INTO game_participants (game_id, player_id, rebuy_count, joined_at)
		VALUES (?, ?, 0, ?)`,
		p.GameID,
		p.PlayerID,
		time.Now().UTC(),
	)
	if err != nil {
		if isUniqueConstraintErr(err) {
			return domain.ErrAlreadyJoined
		}
		return fmt.Errorf("storage.ParticipantRepo.Join: %w", err)
	}
	return nil
}

// IncrementRebuy adds 1 to rebuy_count for a participant.
func (r *ParticipantRepo) IncrementRebuy(ctx context.Context, gameID, playerID int64) error {
	q := extractDB(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE game_participants SET rebuy_count = rebuy_count + 1 WHERE game_id = ? AND player_id = ?`,
		gameID, playerID,
	)
	if err != nil {
		return fmt.Errorf("storage.ParticipantRepo.IncrementRebuy: %w", err)
	}
	return nil
}

// DecrementRebuy subtracts 1 from rebuy_count, but never below 0.
func (r *ParticipantRepo) DecrementRebuy(ctx context.Context, gameID, playerID int64) error {
	q := extractDB(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE game_participants SET rebuy_count = MAX(0, rebuy_count - 1) WHERE game_id = ? AND player_id = ?`,
		gameID, playerID,
	)
	if err != nil {
		return fmt.Errorf("storage.ParticipantRepo.DecrementRebuy: %w", err)
	}
	return nil
}

// ListByGame returns all participants for a game.
func (r *ParticipantRepo) ListByGame(ctx context.Context, gameID int64) ([]domain.Participant, error) {
	q := extractDB(ctx, r.db)
	rows, err := q.QueryContext(ctx, `
		SELECT id, game_id, player_id, rebuy_count, final_chips, results_confirmed, joined_at
		FROM game_participants WHERE game_id = ?`, gameID)
	if err != nil {
		return nil, fmt.Errorf("storage.ParticipantRepo.ListByGame: %w", err)
	}
	defer rows.Close()

	var participants []domain.Participant
	for rows.Next() {
		var p domain.Participant
		if err := rows.Scan(
			&p.ID,
			&p.GameID,
			&p.PlayerID,
			&p.RebuyCount,
			&p.FinalChips,
			&p.ResultsConfirmed,
			&p.JoinedAt,
		); err != nil {
			return nil, fmt.Errorf("storage.ParticipantRepo.ListByGame scan: %w", err)
		}
		participants = append(participants, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage.ParticipantRepo.ListByGame rows: %w", err)
	}
	return participants, nil
}

// SetFinalChips records the final chip count for a participant.
func (r *ParticipantRepo) SetFinalChips(ctx context.Context, gameID, playerID int64, chips int64) error {
	q := extractDB(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE game_participants SET final_chips = ? WHERE game_id = ? AND player_id = ?`,
		chips, gameID, playerID,
	)
	if err != nil {
		return fmt.Errorf("storage.ParticipantRepo.SetFinalChips: %w", err)
	}
	return nil
}

// SetResultsConfirmed marks a participant's results as confirmed.
func (r *ParticipantRepo) SetResultsConfirmed(ctx context.Context, gameID, playerID int64) error {
	q := extractDB(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE game_participants SET results_confirmed = 1 WHERE game_id = ? AND player_id = ?`,
		gameID, playerID,
	)
	if err != nil {
		return fmt.Errorf("storage.ParticipantRepo.SetResultsConfirmed: %w", err)
	}
	return nil
}

// GetByGameAndPlayer fetches a single participant. Returns domain.ErrNotFound if absent.
func (r *ParticipantRepo) GetByGameAndPlayer(ctx context.Context, gameID, playerID int64) (*domain.Participant, error) {
	q := extractDB(ctx, r.db)
	row := q.QueryRowContext(ctx, `
		SELECT id, game_id, player_id, rebuy_count, final_chips, results_confirmed, joined_at
		FROM game_participants WHERE game_id = ? AND player_id = ?`, gameID, playerID)

	var p domain.Participant
	err := row.Scan(
		&p.ID,
		&p.GameID,
		&p.PlayerID,
		&p.RebuyCount,
		&p.FinalChips,
		&p.ResultsConfirmed,
		&p.JoinedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("storage.ParticipantRepo.GetByGameAndPlayer: %w", err)
	}
	return &p, nil
}

// ResetResultsConfirmed sets results_confirmed=false for a participant.
func (r *ParticipantRepo) ResetResultsConfirmed(ctx context.Context, gameID, playerID int64) error {
	q := extractDB(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE game_participants SET results_confirmed = 0 WHERE game_id = ? AND player_id = ?`,
		gameID, playerID,
	)
	if err != nil {
		return fmt.Errorf("storage.ParticipantRepo.ResetResultsConfirmed: %w", err)
	}
	return nil
}

// isUniqueConstraintErr reports whether err is a SQLite UNIQUE constraint violation.
func isUniqueConstraintErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}
