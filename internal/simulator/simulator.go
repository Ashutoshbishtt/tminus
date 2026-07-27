// Package simulator generates the load everything else is measured against. It
// runs thousands of virtual riders moving around the city on their own goroutines
// and creates orders in realistic shapes: steady, lunch peak, dinner peak, and
// sudden spikes. Seeded, so the same run can be repeated and compared.
//
// Built across days 35 to 38.
package simulator

import (
	"context"
	"log/slog"

	"github.com/Ashutoshbishtt/tminus/internal/platform/config"
)

// Run drives the simulated load until the context is cancelled, then stops every
// virtual rider and writes out what happened.
func Run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	log.Info("started")

	<-ctx.Done()

	// Draining means stopping every virtual rider and writing out what happened,
	// so the invariant checker has a complete record to work from.
	log.Info("draining")
	return nil
}
