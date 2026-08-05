// Package db opens the connection pool every service uses to reach Postgres.
// See ADR 0008 for why pgx.
package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Ashutoshbishtt/tminus/internal/platform/config"
)

// Open builds the pool and pings it once, so a service that cannot reach the
// database fails at startup rather than on its first request. The caller owns
// the pool and must Close it.
func Open(ctx context.Context, cfg config.Config, log *slog.Logger) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.PostgresURL)
	if err != nil {
		// The error text can quote the URL back, and the URL holds a password.
		return nil, errors.New("TMINUS_POSTGRES_URL: pgx could not parse it")
	}

	p := cfg.Postgres

	// Settings pgx enforces itself, on its own sockets.
	poolCfg.MaxConns = p.MaxConns
	poolCfg.MinConns = p.MinConns
	poolCfg.MaxConnLifetime = p.MaxConnLifetime
	poolCfg.MaxConnIdleTime = p.MaxConnIdleTime
	poolCfg.ConnConfig.ConnectTimeout = p.ConnectTimeout

	// Settings Postgres enforces, sent when each connection opens. statement_timeout
	// is the only one that stops the server working on a query the client has
	// already given up waiting for.
	poolCfg.ConnConfig.RuntimeParams["statement_timeout"] = millis(p.StatementTimeout)
	poolCfg.ConnConfig.RuntimeParams["lock_timeout"] = millis(p.LockTimeout)
	poolCfg.ConnConfig.RuntimeParams["application_name"] = "tminus-" + cfg.Service

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("postgres pool: %w", err)
	}

	// NewWithConfig is lazy. Force one connection now so a bad password or a
	// database that is down is found here, not on the first request.
	pingCtx, cancel := context.WithTimeout(ctx, p.ConnectTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres is not answering: %w", err)
	}

	log.Info("postgres connected",
		"max_conns", p.MaxConns,
		"min_conns", p.MinConns,
		"statement_timeout", p.StatementTimeout,
	)
	return pool, nil
}

// millis renders a duration the way Postgres wants its timeout settings.
func millis(d time.Duration) string {
	return strconv.FormatInt(d.Milliseconds(), 10)
}
