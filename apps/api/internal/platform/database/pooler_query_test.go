package database_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The exec mode used behind a pooler must still be able to send an array.
//
// There is a production outage behind this. Behind a transaction pooler the
// mode cannot be pgx's default, or prepared statement names collide between
// clients. The first attempt used QueryExecModeExec, which skips asking the
// server what the parameters are — leaving pgx only the Go type to go on, and
// `[]uuid.UUID` for `= ANY($1)` defeats it: "unable to encode ... into text
// format for unknown type (OID 0)". Every query loading checklists failed,
// which is most of the events screen.
//
// DescribeExec asks first, so it knows the parameter is uuid[]. This pins
// both halves: the mode we use works, and the one we replaced does not — so
// anybody tempted to save the round trip finds out here.
func TestExecModes_ArrayParameter(t *testing.T) {
	url := os.Getenv("AMORAE_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("AMORAE_TEST_DATABASE_URL not set; skipping integration test")
	}
	ctx := context.Background()
	ids := []uuid.UUID{uuid.New(), uuid.New()}

	run := func(t *testing.T, mode pgx.QueryExecMode) error {
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
			`SELECT count(*) FROM (SELECT unnest($1::uuid[]) AS id) t`, ids,
		).Scan(&got)
		if err == nil && got != len(ids) {
			t.Errorf("count = %d, want %d", got, len(ids))
		}
		return err
	}

	t.Run("DescribeExec carries a []uuid.UUID", func(t *testing.T) {
		if err := run(t, pgx.QueryExecModeDescribeExec); err != nil {
			t.Fatalf("the mode used behind a pooler refused an array: %v", err)
		}
	})

	t.Run("Exec cannot, which is why it is not used", func(t *testing.T) {
		if err := run(t, pgx.QueryExecModeExec); err == nil {
			t.Skip("this pgx version can encode it; the comment in Connect can be revisited")
		}
	})
}
