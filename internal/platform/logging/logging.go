// Package logging builds the logger every service uses.
//
// Structured logging, so a line is a set of key and value pairs rather than a
// sentence I have to grep. Text while developing because it is readable by eye,
// JSON everywhere else because it is read by machines.
package logging

import (
	"io"
	"log/slog"

	"github.com/Ashutoshbishtt/tminus/internal/platform/config"
)

// New builds a logger for a service. Every line it writes carries the service
// name, which matters as soon as more than one service is running.
func New(w io.Writer, cfg config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}

	var h slog.Handler
	if cfg.LogFormat == "json" {
		h = slog.NewJSONHandler(w, opts)
	} else {
		h = slog.NewTextHandler(w, opts)
	}

	return slog.New(h).With(slog.String("service", cfg.Service))
}
