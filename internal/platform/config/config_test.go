package config

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

// clearEnv makes a test see the environment a fresh clone sees, whatever the
// shell or the CI runner happens to have exported. Blank counts as not set, so
// this is enough to hide a real value, and t.Setenv puts everything back after.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"TMINUS_ENV",
		"TMINUS_LOG_LEVEL",
		"TMINUS_LOG_FORMAT",
		"TMINUS_SHUTDOWN_TIMEOUT",
		"TMINUS_HTTP_ADDR",
		"TMINUS_POSTGRES_URL",
		"TMINUS_REDIS_ADDR",
		"TMINUS_KAFKA_BROKERS",
		"TMINUS_RABBIT_URL",
	} {
		t.Setenv(key, "")
	}
}

func TestLoadDefaults(t *testing.T) {
	// No environment variables set, so this is what a fresh clone gets. These
	// defaults have to match docker-compose.yml or `make run` breaks on day one.
	clearEnv(t)

	cfg, err := Load("api")
	if err != nil {
		t.Fatalf("Load with no environment set: %v", err)
	}

	checks := []struct {
		name string
		got  any
		want any
	}{
		{"service", cfg.Service, "api"},
		{"env", cfg.Env, "local"},
		{"log level", cfg.LogLevel, slog.LevelInfo},
		{"log format", cfg.LogFormat, "text"},
		{"shutdown timeout", cfg.ShutdownTimeout, 15 * time.Second},
		{"http addr", cfg.HTTPAddr, ":8000"},
		{"redis addr", cfg.RedisAddr, "localhost:6379"},
		{"kafka brokers", len(cfg.KafkaBrokers), 1},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestLoadRejectsBadSettings(t *testing.T) {
	tests := []struct {
		name string
		key  string
		val  string
	}{
		{"log level that is not a level", "TMINUS_LOG_LEVEL", "chatty"},
		{"log format we do not have", "TMINUS_LOG_FORMAT", "xml"},
		{"shutdown timeout that is not a duration", "TMINUS_SHUTDOWN_TIMEOUT", "soon"},
		{"negative shutdown timeout", "TMINUS_SHUTDOWN_TIMEOUT", "-5s"},
		{"postgres url with the wrong scheme", "TMINUS_POSTGRES_URL", "mysql://tminus@localhost:3306/tminus"},
		{"rabbit url with the wrong scheme", "TMINUS_RABBIT_URL", "http://localhost:5672"},
		{"kafka brokers set to nothing", "TMINUS_KAFKA_BROKERS", ","},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear first, so a failure is caused by the value under test and not
			// by something else broken that happened to be in the environment.
			clearEnv(t)
			t.Setenv(tt.key, tt.val)

			if _, err := Load("api"); err == nil {
				t.Fatalf("%s=%q was accepted, expected it to be refused at startup", tt.key, tt.val)
			}
		})
	}
}

func TestBlankMeansNotSet(t *testing.T) {
	// A variable that is present but blank, or is just spaces, should be treated
	// as if nobody set it. Quietly accepting " " as an address is worse than
	// falling back to something that works.
	clearEnv(t)
	t.Setenv("TMINUS_HTTP_ADDR", "  ")
	t.Setenv("TMINUS_REDIS_ADDR", "")

	cfg, err := Load("api")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HTTPAddr != ":8000" {
		t.Errorf("http addr: got %q, want the default :8000", cfg.HTTPAddr)
	}
	if cfg.RedisAddr != "localhost:6379" {
		t.Errorf("redis addr: got %q, want the default localhost:6379", cfg.RedisAddr)
	}
}

func TestKafkaBrokersAreSplitAndTrimmed(t *testing.T) {
	t.Setenv("TMINUS_KAFKA_BROKERS", " one:9092 , two:9092,three:9092 ")

	cfg, err := Load("locator")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := []string{"one:9092", "two:9092", "three:9092"}
	if len(cfg.KafkaBrokers) != len(want) {
		t.Fatalf("got %v, want %v", cfg.KafkaBrokers, want)
	}
	for i := range want {
		if cfg.KafkaBrokers[i] != want[i] {
			t.Errorf("broker %d: got %q, want %q", i, cfg.KafkaBrokers[i], want[i])
		}
	}
}

func TestPasswordsAreNeverLogged(t *testing.T) {
	const secret = "hunter2"
	t.Setenv("TMINUS_POSTGRES_URL", "postgres://tminus:"+secret+"@localhost:5432/tminus")
	t.Setenv("TMINUS_RABBIT_URL", "amqp://tminus:"+secret+"@localhost:5672/")

	cfg, err := Load("api")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	logged := cfg.LogValue().String()
	if strings.Contains(logged, secret) {
		t.Fatalf("the password ended up in the logs: %s", logged)
	}
	// The rest of the URL should survive, otherwise the log line is useless.
	if !strings.Contains(logged, "localhost:5432") {
		t.Errorf("redaction removed too much, host is missing: %s", logged)
	}
}

func TestRedactLeavesURLsWithoutPasswordsAlone(t *testing.T) {
	const plain = "postgres://tminus@localhost:5432/tminus"
	if got := redact(plain); got != plain {
		t.Errorf("got %q, want it unchanged", got)
	}
}
