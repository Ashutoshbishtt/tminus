// Command tminus runs every service in the project. Which one it becomes is
// decided by the first argument, so there is one binary to build and version
// instead of five.
//
//	tminus api          the REST and WebSocket endpoints
//	tminus dispatcher   the dispatch engine
//	tminus locator      keeps the nearby-rider index fresh
//	tminus notifier     sends notifications
//	tminus simulator    generates load
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/Ashutoshbishtt/tminus/internal/api"
	"github.com/Ashutoshbishtt/tminus/internal/dispatcher"
	"github.com/Ashutoshbishtt/tminus/internal/locator"
	"github.com/Ashutoshbishtt/tminus/internal/notifier"
	"github.com/Ashutoshbishtt/tminus/internal/platform/config"
	"github.com/Ashutoshbishtt/tminus/internal/platform/logging"
	"github.com/Ashutoshbishtt/tminus/internal/simulator"
)

// version is set at build time from the Makefile.
var version = "dev"

// runFunc is the shape every service shares. main hands over a context, the
// settings and a logger, then waits. It does not care which service it started.
type runFunc func(ctx context.Context, cfg config.Config, log *slog.Logger) error

var services = map[string]runFunc{
	"api":        api.Run,
	"dispatcher": dispatcher.Run,
	"locator":    locator.Run,
	"notifier":   notifier.Run,
	"simulator":  simulator.Run,
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "tminus:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no service given\n\n%s", usage())
	}
	name := args[0]
	service, ok := services[name]
	if !ok {
		return fmt.Errorf("unknown service %q\n\n%s", name, usage())
	}

	cfg, err := config.Load(name)
	if err != nil {
		return err
	}
	log := logging.New(os.Stdout, cfg)

	// A stop signal cancels this context, and everything the service does hangs
	// off it. So one Ctrl-C reaches all the way down to whatever is holding a
	// connection open.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("starting", "version", version, "config", cfg)

	// stop is passed in so that the moment a signal is seen, the default signal
	// behaviour is put back and a second Ctrl-C kills the process outright,
	// instead of waiting politely alongside me.
	return supervise(ctx, stop, service, cfg, log)
}

// supervise runs a service and sees it out. It returns once the service has
// stopped, or once it has been given long enough to stop and has not.
func supervise(ctx context.Context, onStopSignal func(), service runFunc, cfg config.Config, log *slog.Logger) error {
	done := make(chan error, 1)
	go func() { done <- service(ctx, cfg, log) }()

	select {
	case err := <-done:
		// Stopped on its own, without being asked. Either it finished its work or
		// it fell over, and either way there is nothing to drain.
		if err != nil {
			return err
		}
		log.Info("stopped")
		return nil

	case <-ctx.Done():
		onStopSignal()
		log.Info("stop signal received, draining", "timeout", cfg.ShutdownTimeout)
	}

	select {
	case err := <-done:
		// context.Canceled here just means the service noticed the cancel and
		// stopped, which is what was asked of it. It is not a failure.
		if err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("during shutdown: %w", err)
		}
		log.Info("stopped cleanly")
		return nil

	case <-time.After(cfg.ShutdownTimeout):
		// Out of patience. Say so plainly: whatever was in flight may be half done,
		// and something upstream will have to redeliver it.
		return fmt.Errorf("did not finish draining within %s, in-flight work may have been dropped", cfg.ShutdownTimeout)
	}
}

func usage() string {
	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	slices.Sort(names)

	var b strings.Builder
	b.WriteString("usage: tminus <service>\n\nservices:\n")
	for _, name := range names {
		fmt.Fprintf(&b, "  %s\n", name)
	}
	return b.String()
}
