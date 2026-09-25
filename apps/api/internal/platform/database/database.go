// Package database wraps pgxpool behind a small interface (Querier) so that
// repositories never depend on *pgxpool.Pool directly, and behind a
// tx-in-context helper (InTx) so a service can compose several repository
// calls into one transaction without any repository knowing whether it's
// inside one.
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

	// Behind a transaction pooler, never cache prepared statements.
	//
	// pgx's default keeps server-side prepared statements and reuses them by
	// name. A transaction pooler — Neon's "-pooler" endpoint, Supabase's,
	// PgBouncer generally — hands the same server connection to different
	// clients between statements, so a name one client prepared turns up
	// already taken for the next: "prepared statement name is already in use"
	// (SQLSTATE 08P01).
	//
	// This helps and does not cure. Read the next comment before trusting it.
	if isTransactionPooler(url, cfg.ConnConfig.Host) {
		cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeDescribeExec
	}

	// DescribeExec is not safe behind a transaction pooler either, and saying
	// otherwise here cost a second outage (2026-09-25). It sends two protocol
	// exchanges — Parse+Describe+Sync, then Bind+Execute+Sync — and a pooler
	// is free to hand those to different backends. The Bind then lands on a
	// server whose unnamed statement belongs to somebody else's query:
	// "bind message supplies 1 parameters, but prepared statement \"\" requires 3".
	// Every signed-in request failed, because session lookup is one of them.
	//
	// The modes that survive a pooler send one exchange, and neither can
	// encode a []uuid.UUID without asking the server what the parameter is.
	// Measured against Postgres rather than reasoned about (see
	// pooler_query_test.go):
	//
	//	Exec           + []uuid.UUID   unable to encode ... unknown type (OID 0)
	//	SimpleProtocol + []uuid.UUID   the same
	//	Exec           + []string      works
	//	SimpleProtocol + []string      works
	//
	// So a pooler can be made to work, by using Exec and passing uuid arrays
	// as strings at the six `= ANY($1)` call sites. That work is not done,
	// because the answer for this application is the other URL: it has its
	// own connection pool and does not need a second one. The line logged at
	// startup says so, loudly, rather than leaving it to be discovered in
	// production a third time.

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
// pooler, for callers that want to say something about it before serving.
// A URL that will not parse is not a pooler; Connect reports that properly.
func IsTransactionPooler(url string) bool {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return false
	}
	return isTransactionPooler(url, cfg.ConnConfig.Host)
}

// isTransactionPooler reports whether this connection goes through one,
// by the two signals that are actually available: the host naming itself, and
// the parameter poolers conventionally take.
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
