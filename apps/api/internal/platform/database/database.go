// Package database wraps pgxpool behind a small interface (Querier), so
// repositories never depend on *pgxpool.Pool directly, and behind a
// tx-in-context helper (InTx), so services can compose repository calls
// into one transaction without any repository knowing it's inside one.
package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier is the subset of *pgxpool.Pool (and *pgx.Tx) that repositories
// need; it's what lets DB.Q return either the pool or an in-flight
// transaction without callers telling the difference.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type DB struct {
	pool *pgxpool.Pool
}

// Connect opens a pool and pings it before returning, so an unreachable
// database fails at startup, not on the first request.
func Connect(ctx context.Context, url string, maxConns int32) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parsing database url: %w", err)
	}
	cfg.MaxConns = maxConns

	// Behind a transaction pooler, pgx's cached prepared statements collide
	// across clients sharing a backend (SQLSTATE 08P01). DescribeExec avoids
	// that, but is still not fully pooler-safe: its two protocol round trips
	// (Parse+Describe+Sync, then Bind+Execute+Sync) can land on different
	// backends, and it can't encode []uuid.UUID without a server round trip
	// (see pooler_query_test.go). This app has its own connection pool and
	// doesn't need a second one — the real fix is the direct DATABASE_URL,
	// not a pooler-safe query mode; this is a mitigation only, and Connect's
	// caller is expected to warn loudly when it's in play.
	if isTransactionPooler(url, cfg.ConnConfig.Host) {
		cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeDescribeExec
	}

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

// IsTransactionPooler reports whether this URL goes through a transaction
// pooler. A URL that won't parse is reported as not one.
func IsTransactionPooler(url string) bool {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return false
	}
	return isTransactionPooler(url, cfg.ConnConfig.Host)
}

// isTransactionPooler checks the two available signals: the host naming
// itself, and the parameter poolers conventionally take.
func isTransactionPooler(url, host string) bool {
	return strings.Contains(host, "-pooler.") ||
		strings.Contains(host, "pgbouncer") ||
		strings.Contains(url, "pgbouncer=true")
}

func (db *DB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

func (db *DB) Close() {
	db.pool.Close()
}

type txKey struct{}

// Q returns the Querier for this ctx: an in-flight transaction if InTx
// started one higher up the call stack, otherwise the pool.
func (db *DB) Q(ctx context.Context) Querier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return db.pool
}

// InTx runs fn inside a transaction, reusing one already on ctx (pgx
// transactions aren't reentrant) so nested calls still get one commit/rollback.
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
