package db_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Ashutoshbishtt/tminus/internal/platform/config"
	"github.com/Ashutoshbishtt/tminus/internal/platform/db"
)

// These need a real Postgres. `make up` provides one; CI runs one as a service.
// They skip rather than fail when there is none, so `go test ./...` still passes
// on a laptop with nothing running.

func quiet() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func open(t *testing.T) (*pgxpool.Pool, context.Context) {
	t.Helper()

	cfg, err := config.Load("test")
	if err != nil {
		t.Fatalf("config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	pool, err := db.Open(ctx, cfg, quiet())
	if err != nil {
		t.Skipf("no postgres to talk to, skipping: %v", err)
	}
	t.Cleanup(pool.Close)

	return pool, ctx
}

func TestOpenConnects(t *testing.T) {
	pool, ctx := open(t)

	var one int
	if err := pool.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
		t.Fatalf("querying: %v", err)
	}
	if one != 1 {
		t.Errorf("got %d, want 1", one)
	}
}

func TestOpenRejectsAnUnreachableDatabase(t *testing.T) {
	// Proves Open pings rather than handing back a pool that has never connected.
	// Without the ping this would succeed here and fail on the first real query.
	cfg, err := config.Load("test")
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	cfg.PostgresURL = "postgres://tminus:tminus@localhost:9999/tminus?sslmode=disable"
	cfg.Postgres.ConnectTimeout = 2 * time.Second

	pool, err := db.Open(context.Background(), cfg, quiet())
	if err == nil {
		pool.Close()
		t.Fatal("Open succeeded against a port with nothing on it")
	}
}

func TestStatementTimeoutIsEnforcedByPostgres(t *testing.T) {
	// The setting that stops the server working on a query the client has already
	// abandoned. Checking it is really in force, not just spelled correctly.
	pool, ctx := open(t)

	var setting string
	if err := pool.QueryRow(ctx, "SHOW statement_timeout").Scan(&setting); err != nil {
		t.Fatalf("reading statement_timeout: %v", err)
	}
	if setting == "0" || setting == "" {
		t.Fatalf("statement_timeout is %q, so a runaway query would never be cut off", setting)
	}

	// And that it actually cuts a query off.
	if _, err := pool.Exec(ctx, "SET statement_timeout = 100"); err != nil {
		t.Fatalf("setting a short timeout: %v", err)
	}
	if _, err := pool.Exec(ctx, "SELECT pg_sleep(5)"); err == nil {
		t.Error("a five second sleep finished under a 100ms timeout")
	}
}
