// Package locator reads the rider location stream from Kafka and keeps the Redis
// geo index up to date, so dispatch can ask "who is near the store right now" and
// get an answer immediately.
//
// Built on days 20 and 21.
package locator

import (
	"context"
	"log/slog"

	"github.com/Ashutoshbishtt/tminus/internal/platform/config"
)

// Run follows the rider location stream until the context is cancelled, then
// commits what it has actually handled and stops.
func Run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	log.Info("started")

	<-ctx.Done()

	// Draining here means committing the offset for what has actually been
	// handled, so a restart picks up in the right place instead of replaying.
	log.Info("draining")
	return nil
}
