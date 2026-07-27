// Package dispatcher is the heart of the project. It pulls dispatch jobs off
// RabbitMQ, finds candidate riders near the store, scores them, offers the order
// to one rider at a time, and assigns the winner under a database constraint that
// makes two riders for one order impossible.
//
// Built across days 22 to 29.
package dispatcher

import (
	"context"
	"log/slog"

	"github.com/Ashutoshbishtt/tminus/internal/platform/config"
)

// Run works through dispatch jobs until the context is cancelled, then finishes
// the job in hand and stops.
func Run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	log.Info("started")

	<-ctx.Done()

	// Once this is consuming jobs, draining means finishing the job in hand and
	// acking it, then stopping. Anything not acked goes back to the queue.
	log.Info("draining")
	return nil
}
