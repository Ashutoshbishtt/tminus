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
	return checkURL("TMINUS_RABBIT_URL", c.RabbitURL, "amqp", "amqps")
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
