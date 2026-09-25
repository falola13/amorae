package database_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// What each exec mode can encode — and, just as importantly, what this file
// does NOT prove.
//
// It does not prove any mode is safe behind a transaction pooler. That needs
// a pooler in front of Postgres, splitting protocol exchanges across backends
// under connection reuse, and there isn't one here. An earlier version of this
// file read as though it did, because it was named for the pooler and pinned
// the mode the pooler path had chosen. The mode passed, and the pooler path
// was broken anyway: DescribeExec sends Parse+Describe+Sync and then
// Bind+Execute+Sync, and a pooler may answer those from different servers —
// "bind message supplies 1 parameters, but prepared statement "" requires 3"
// (SQLSTATE 08P01), which took every signed-in request down on 2026-09-25.
//
// So: a green run here means arrays encode. It means nothing about poolers.
// The answer for poolers is not to use one (see Connect).
//
// What the encoding results are good for is knowing the way out if a pooler
// ever becomes unavoidable. The modes that survive one send a single
// exchange, and neither can type a []uuid.UUID without asking the server —
// but both manage it once the ids are strings, which is a change at six
// `= ANY($1)` call sites rather than a redesign.
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

	// The mode the code actually uses on a direct connection's pooler branch.
	// It encodes arrays, which is the only claim being made for it here.
	t.Run("DescribeExec carries a []uuid.UUID", func(t *testing.T) {
		if err := run(t, pgx.QueryExecModeDescribeExec, ids); err != nil {
			t.Fatalf("DescribeExec refused an array: %v", err)
		}
	})

	// Both single-exchange modes need the server's help to type a uuid array
	// and do not get it. This is why the first pooler fix took the events
	// screen down: "unable to encode ... for unknown type (OID 0)".
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

	// ...but they manage the same query once the ids are strings. This is the
	// documented way out, and it is here so that the claim is measured rather
	// than remembered.
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
