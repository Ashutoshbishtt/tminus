// Package config holds every setting the services need, read once at startup.
//
// Everything comes from environment variables, and every one has a default that
// points at the local stack from docker-compose.yml. That means `make run` works
// on a fresh clone with no setup. Anything wrong is caught here, before the
// service starts doing work.
package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the whole configuration of any tminus service. One struct for all
// five, because they share most of it and the extra fields cost nothing.
type Config struct {
	// Service is which role this process is running as: api, dispatcher, and so on.
	Service string
	// Env is local, staging or production. Only used for logging and for choosing
	// sensible defaults.
	Env string

	LogLevel  slog.Level
	LogFormat string

	// ShutdownTimeout is how long to wait for in-flight work to finish after a
	// stop signal before giving up. Keep it under whatever the container runtime
	// allows, or the process gets killed mid-drain.
	ShutdownTimeout time.Duration

	HTTPAddr     string
	PostgresURL  string
	RedisAddr    string
	KafkaBrokers []string
	RabbitURL    string

	Postgres PostgresPool
}

// PostgresPool is how much of the database one service may use. Postgres allows
// 100 connections and five services share them, so each takes a slice rather
// than opening whatever it likes.
type PostgresPool struct {
	MaxConns int32
	MinConns int32

	ConnectTimeout   time.Duration
	StatementTimeout time.Duration
	LockTimeout      time.Duration

	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// Load reads the settings for a service and checks them. It returns an error
// rather than exiting, so the caller decides how to report it.
func Load(service string) (Config, error) {
	cfg := Config{
		Service:     service,
		Env:         env("TMINUS_ENV", "local"),
		LogFormat:   env("TMINUS_LOG_FORMAT", "text"),
		HTTPAddr:    env("TMINUS_HTTP_ADDR", ":8000"),
		PostgresURL: env("TMINUS_POSTGRES_URL", "postgres://tminus:tminus@localhost:5432/tminus?sslmode=disable"),
		RedisAddr:   env("TMINUS_REDIS_ADDR", "localhost:6379"),
		RabbitURL:   env("TMINUS_RABBIT_URL", "amqp://tminus:tminus@localhost:5672/"),
	}

	for _, b := range strings.Split(env("TMINUS_KAFKA_BROKERS", "localhost:9092"), ",") {
		if b = strings.TrimSpace(b); b != "" {
			cfg.KafkaBrokers = append(cfg.KafkaBrokers, b)
		}
	}

	level, err := parseLevel(env("TMINUS_LOG_LEVEL", "info"))
	if err != nil {
		return Config{}, err
	}
	cfg.LogLevel = level

	timeout, err := time.ParseDuration(env("TMINUS_SHUTDOWN_TIMEOUT", "15s"))
	if err != nil {
		return Config{}, fmt.Errorf("TMINUS_SHUTDOWN_TIMEOUT: %w", err)
	}
	cfg.ShutdownTimeout = timeout

	pool, err := loadPostgresPool()
	if err != nil {
		return Config{}, err
	}
	cfg.Postgres = pool

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// validate catches the mistakes that would otherwise show up much later, as a
// confusing failure in the middle of a run.
func (c Config) validate() error {
	switch c.LogFormat {
	case "text", "json":
	default:
		return fmt.Errorf("TMINUS_LOG_FORMAT: want text or json, got %q", c.LogFormat)
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("TMINUS_SHUTDOWN_TIMEOUT: must be positive, got %s", c.ShutdownTimeout)
	}
	if c.HTTPAddr == "" {
		return fmt.Errorf("TMINUS_HTTP_ADDR: must not be empty")
	}
	if c.RedisAddr == "" {
		return fmt.Errorf("TMINUS_REDIS_ADDR: must not be empty")
	}
	if len(c.KafkaBrokers) == 0 {
		return fmt.Errorf("TMINUS_KAFKA_BROKERS: must list at least one broker")
	}
	if err := checkURL("TMINUS_POSTGRES_URL", c.PostgresURL, "postgres", "postgresql"); err != nil {
		return err
	}
	if err := c.Postgres.validate(); err != nil {
		return err
	}
	return checkURL("TMINUS_RABBIT_URL", c.RabbitURL, "amqp", "amqps")
}

func (p PostgresPool) validate() error {
	if p.MaxConns < 1 {
		return fmt.Errorf("TMINUS_POSTGRES_MAX_CONNS: must be at least 1, got %d", p.MaxConns)
	}
	if p.MinConns < 0 {
		return fmt.Errorf("TMINUS_POSTGRES_MIN_CONNS: must not be negative, got %d", p.MinConns)
	}
	// pgxpool rejects this at connect time anyway, but then the message comes out
	// of a library instead of naming the two variables that disagree.
	if p.MinConns > p.MaxConns {
		return fmt.Errorf("TMINUS_POSTGRES_MIN_CONNS (%d) must not exceed TMINUS_POSTGRES_MAX_CONNS (%d)", p.MinConns, p.MaxConns)
	}
	for _, d := range []struct {
		key string
		val time.Duration
	}{
		{"TMINUS_POSTGRES_CONNECT_TIMEOUT", p.ConnectTimeout},
		{"TMINUS_POSTGRES_STATEMENT_TIMEOUT", p.StatementTimeout},
		{"TMINUS_POSTGRES_LOCK_TIMEOUT", p.LockTimeout},
		{"TMINUS_POSTGRES_MAX_CONN_LIFETIME", p.MaxConnLifetime},
		{"TMINUS_POSTGRES_MAX_CONN_IDLE_TIME", p.MaxConnIdleTime},
	} {
		if d.val <= 0 {
			return fmt.Errorf("%s: must be positive, got %s", d.key, d.val)
		}
	}
	return nil
}

// LogValue lets a whole Config be logged in one go without leaking passwords.
// slog calls this instead of reflecting over the struct.
func (c Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("env", c.Env),
		slog.String("log_level", c.LogLevel.String()),
		// As a string, because slog writes a Duration to JSON as a bare nanosecond
		// count and 15000000000 is nobody's idea of readable.
		slog.String("shutdown_timeout", c.ShutdownTimeout.String()),
		slog.String("http_addr", c.HTTPAddr),
		slog.String("postgres", redact(c.PostgresURL)),
		slog.String("redis", c.RedisAddr),
		slog.String("kafka", strings.Join(c.KafkaBrokers, ",")),
		slog.String("rabbit", redact(c.RabbitURL)),
		slog.Group("pg_pool",
			slog.Int("max_conns", int(c.Postgres.MaxConns)),
			slog.Int("min_conns", int(c.Postgres.MinConns)),
			slog.String("connect_timeout", c.Postgres.ConnectTimeout.String()),
			slog.String("statement_timeout", c.Postgres.StatementTimeout.String()),
		),
	)
}

// env reads a setting, treating blank as not set at all. Surrounding spaces are
// trimmed, because they are almost always a typo in a compose file or a shell
// export, and a value of " " should not quietly become a real setting.
func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return fallback
}

// loadPostgresPool reads the pool settings. The defaults are worked out in
// learning/notes/connection-pool.md: 100 connections in Postgres, five services
// sharing them, so 10 each leaves room for psql and a migration run.
func loadPostgresPool() (PostgresPool, error) {
	var p PostgresPool
	var err error

	if p.MaxConns, err = envInt32("TMINUS_POSTGRES_MAX_CONNS", 10); err != nil {
		return p, err
	}
	if p.MinConns, err = envInt32("TMINUS_POSTGRES_MIN_CONNS", 2); err != nil {
		return p, err
	}
	if p.ConnectTimeout, err = envDuration("TMINUS_POSTGRES_CONNECT_TIMEOUT", "5s"); err != nil {
		return p, err
	}
	if p.StatementTimeout, err = envDuration("TMINUS_POSTGRES_STATEMENT_TIMEOUT", "30s"); err != nil {
		return p, err
	}
	if p.LockTimeout, err = envDuration("TMINUS_POSTGRES_LOCK_TIMEOUT", "5s"); err != nil {
		return p, err
	}
	if p.MaxConnLifetime, err = envDuration("TMINUS_POSTGRES_MAX_CONN_LIFETIME", "1h"); err != nil {
		return p, err
	}
	if p.MaxConnIdleTime, err = envDuration("TMINUS_POSTGRES_MAX_CONN_IDLE_TIME", "30m"); err != nil {
		return p, err
	}
	return p, nil
}

func envDuration(key, fallback string) (time.Duration, error) {
	d, err := time.ParseDuration(env(key, fallback))
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return d, nil
}

func envInt32(key string, fallback int32) (int32, error) {
	raw := env(key, strconv.FormatInt(int64(fallback), 10))
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s: want a whole number, got %q", key, raw)
	}
	return int32(n), nil
}

func parseLevel(s string) (slog.Level, error) {
	var l slog.Level
	// UnmarshalText understands debug, info, warn and error, in any case.
	if err := l.UnmarshalText([]byte(s)); err != nil {
		return 0, fmt.Errorf("TMINUS_LOG_LEVEL: want debug, info, warn or error, got %q", s)
	}
	return l, nil
}

func checkURL(key, raw string, schemes ...string) error {
	u, err := url.Parse(raw)
	if err != nil {
		// The error text can contain the URL, and the URL can contain a password.
		return fmt.Errorf("%s: not a valid URL", key)
	}
	for _, s := range schemes {
		if u.Scheme == s {
			return nil
		}
	}
	return fmt.Errorf("%s: want scheme %s, got %q", key, strings.Join(schemes, " or "), u.Scheme)
}

// redact swaps a password out of a connection URL so it can be logged. Anything
// that fails to parse is not echoed back, in case the thing that failed to parse
// was the password.
func redact(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "(unparseable)"
	}
	if u.User != nil {
		if _, hasPassword := u.User.Password(); hasPassword {
			u.User = url.UserPassword(u.User.Username(), "xxxxx")
		}
	}
	return u.String()
}
