package database_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Measures what each exec mode can encode. It does NOT prove any mode is
// safe behind a transaction pooler — that needs an actual pooler splitting
// protocol exchanges across backends, and there isn't one here (see Connect
// for why this app avoids poolers rather than chasing that safety).
//
// Useful if a pooler ever becomes unavoidable: the modes that survive one
// send a single exchange but can't type a []uuid.UUID without the server's
// help — passing ids as strings works instead.
func TestExecModes_ArrayParameter(t *testing.T) {
	url := os.Getenv("AMORAE_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("AMORAE_TEST_DATABASE_URL not set; skipping integration test")
	}
	ctx := context.Background()
	ids := []uuid.UUID{uuid.New(), uuid.New()}
	strs := []string{ids[0].String(), ids[1].String()}

	run := func(t *testing.T, mode pgx.QueryExecMode, arg any) error {
		t.Helper()
		cfg, err := pgxpool.ParseConfig(url)
		if err != nil {
			t.Fatalf("parsing url: %v", err)
		}
		cfg.ConnConfig.DefaultQueryExecMode = mode
		pool, err := pgxpool.NewWithConfig(ctx, cfg)
		if err != nil {
			t.Fatalf("connecting: %v", err)
		}
		defer pool.Close()

		var got int
		err = pool.QueryRow(ctx,
			`SELECT count(*) FROM (SELECT unnest($1::uuid[]) AS id) t`, arg,
		).Scan(&got)
		if err == nil && got != len(ids) {
			t.Errorf("count = %d, want %d", got, len(ids))
		}
		return err
	}

	t.Run("DescribeExec carries a []uuid.UUID", func(t *testing.T) {
		if err := run(t, pgx.QueryExecModeDescribeExec, ids); err != nil {
			t.Fatalf("DescribeExec refused an array: %v", err)
		}
	})

	t.Run("the pooler-survivable modes cannot type a uuid array", func(t *testing.T) {
		for name, mode := range map[string]pgx.QueryExecMode{
			"Exec":           pgx.QueryExecModeExec,
			"SimpleProtocol": pgx.QueryExecModeSimpleProtocol,
		} {
			if err := run(t, mode, ids); err == nil {
				t.Errorf("%s encoded a []uuid.UUID; if pgx learned to, Connect's comment can be revisited", name)
			}
		}
	})

	t.Run("and manage it once the ids are strings", func(t *testing.T) {
		for name, mode := range map[string]pgx.QueryExecMode{
			"Exec":           pgx.QueryExecModeExec,
			"SimpleProtocol": pgx.QueryExecModeSimpleProtocol,
		} {
			if err := run(t, mode, strs); err != nil {
				t.Errorf("%s refused a []string: %v", name, err)
			}
		}
	})
}
