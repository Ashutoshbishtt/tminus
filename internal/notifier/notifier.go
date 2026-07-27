// Package notifier takes notification jobs off RabbitMQ and sends them, routing
// each one to the right channel. Failures are retried with a growing delay, and
// the hopeless ones are parked in the dead letter queue.
//
// Built across days 32 to 34.
package notifier

import (
	"context"
	"log/slog"

	"github.com/Ashutoshbishtt/tminus/internal/platform/config"
)

// Run sends notifications until the context is cancelled, then finishes the one
// in hand and stops.
func Run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	log.Info("started")

	<-ctx.Done()

	log.Info("draining")
	return nil
}
