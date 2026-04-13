package service

import (
	"context"
	"time"

	"poker-bot/internal/domain"
)

// TxManager runs fn inside a database transaction. The transaction is injected
// into ctx so that repositories can participate transparently.
type TxManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// PlayerRepository is the persistence interface for Player entities.
type PlayerRepository interface {
	GetByTelegramID(ctx context.Context, telegramID int64) (*domain.Player, error)
	Upsert(ctx context.Context, p *domain.Player) error
}

// GameRepository is the persistence interface for Game entities.
type GameRepository interface {
	Create(ctx context.Context, g *domain.Game) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.Game, error)
	GetActiveByChatID(ctx context.Context, chatID int64) (*domain.Game, error)
	GetCollectingByPlayerID(ctx context.Context, playerID int64) (*domain.Game, error)
	GetFinishedByPlayerID(ctx context.Context, playerID int64) (*domain.Game, error)
	UpdateStatus(ctx context.Context, id int64, status domain.GameStatus) error
	SetHubMessageID(ctx context.Context, id int64, msgID int64) error
	SetFinishedAt(ctx context.Context, id int64, t time.Time) error
}

// ParticipantRepository is the persistence interface for Participant entities.
type ParticipantRepository interface {
	Join(ctx context.Context, p *domain.Participant) error
	IncrementRebuy(ctx context.Context, gameID, playerID int64) error
	DecrementRebuy(ctx context.Context, gameID, playerID int64) error
	ListByGame(ctx context.Context, gameID int64) ([]domain.Participant, error)
	SetFinalChips(ctx context.Context, gameID, playerID int64, chips int64) error
	SetResultsConfirmed(ctx context.Context, gameID, playerID int64) error
	ResetResultsConfirmed(ctx context.Context, gameID, playerID int64) error
	GetByGameAndPlayer(ctx context.Context, gameID, playerID int64) (*domain.Participant, error)
}

// SettlementRepository is the persistence interface for Settlement entities.
type SettlementRepository interface {
	SaveAll(ctx context.Context, gameID int64, transfers []domain.Transfer) error
	ListByGame(ctx context.Context, gameID int64) ([]domain.Settlement, error)
}
