// Package database wraps pgxpool behind a small interface (Querier) so that
// repositories never depend on *pgxpool.Pool directly, and behind a
// tx-in-context helper (InTx) so a service can compose several repository
// calls into one transaction without any repository knowing whether it's
// inside one.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier is the subset of *pgxpool.Pool (and *pgx.Tx, which has the same
// three methods) that repositories need. Declaring it here — rather than
// repositories importing pgxpool directly — is what lets DB.Q return
// either the pool or an in-flight transaction and have callers unable to
// tell the difference.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type DB struct {
	pool *pgxpool.Pool
}

// Connect opens a pool and confirms the database is actually reachable
// before returning — a pool that merely parses is not the same as a
// database that answers, and we want config.Load-time failures to surface
// at startup, not on the first request.
func Connect(ctx context.Context, url string, maxConns int32) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parsing database url: %w", err)
	}
	cfg.MaxConns = maxConns

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &DB{pool: pool}, nil
}

func (db *DB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

func (db *DB) Close() {
	db.pool.Close()
}

type txKey struct{}

// Q returns the Querier a repository should use for this ctx: the
// in-flight transaction if one was started by InTx higher up the call
// stack, otherwise the pool. This is the whole mechanism behind
// repositories "transparently" joining a transaction — they always call
// db.Q(ctx) and never hold a reference to the pool or a tx themselves.
func (db *DB) Q(ctx context.Context) Querier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return db.pool
}

// InTx runs fn inside a transaction. If ctx already carries a transaction
// (an outer InTx call), fn reuses it instead of nesting a second one —
// pgx transactions aren't reentrant, and a service calling two repository
// methods that each want "their own" transaction should still get exactly
// one commit/rollback for the whole operation.
func (db *DB) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	txCtx := context.WithValue(ctx, txKey{}, tx)

	defer func() {
		// A panic mid-transaction must still roll back before it propagates,
		// otherwise the connection goes back to the pool mid-transaction.
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(txCtx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("%w (rollback also failed: %v)", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}
