package service

import (
	"context"

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
