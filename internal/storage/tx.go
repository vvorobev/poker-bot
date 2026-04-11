package storage

import (
	"context"
	"database/sql"
	"fmt"
)

type txKey struct{}

// TxManager implements service.TxManager using *sql.DB.
type TxManager struct {
	db *sql.DB
}

// NewTxManager creates a TxManager backed by db.
func NewTxManager(db *sql.DB) *TxManager {
	return &TxManager{db: db}
}

// RunInTx begins a transaction, injects it into ctx, and calls fn.
// Commits on success, rolls back on error or panic.
func (m *TxManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("storage.RunInTx begin: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	txCtx := context.WithValue(ctx, txKey{}, tx)
	if err := fn(txCtx); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("storage.RunInTx commit: %w", err)
	}
	return nil
}

// extractDB returns the *sql.Tx from ctx if present (as a db-compatible executor),
// otherwise returns the plain *sql.DB. Both implement the querier interface below.
func extractDB(ctx context.Context, db *sql.DB) querier {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return tx
	}
	return db
}

// querier is the minimal interface shared by *sql.DB and *sql.Tx.
type querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}
