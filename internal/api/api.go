// Package api serves everything the outside world talks to: the REST endpoints
// for placing orders and managing riders, and the WebSocket connections that
// carry live offers to riders and live tracking to customers.
//
// The endpoints arrive on day 5. For now it starts and stops.
package api

import (
	"context"
	"log/slog"

	"github.com/Ashutoshbishtt/tminus/internal/platform/config"
)

// Run serves the API until the context is cancelled, then stops taking new
// connections and lets the requests already in flight finish.
func Run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	log.Info("started", "http_addr", cfg.HTTPAddr)

	<-ctx.Done()

	// Once there is an HTTP server here, this is where it stops accepting new
	// connections and lets the in-flight requests finish.
	log.Info("draining")
	return nil
}
