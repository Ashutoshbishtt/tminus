package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/Ashutoshbishtt/tminus/internal/platform/config"
)

func testSetup(t *testing.T, shutdownTimeout time.Duration) (config.Config, *slog.Logger) {
	t.Helper()
	cfg := config.Config{Service: "test", ShutdownTimeout: shutdownTimeout}
	return cfg, slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestServiceThatStopsOnItsOwn(t *testing.T) {
	cfg, log := testSetup(t, time.Second)

	err := supervise(context.Background(), func() {}, func(context.Context, config.Config, *slog.Logger) error {
		return nil
	}, cfg, log)

	if err != nil {
		t.Fatalf("got %v, want no error", err)
	}
}

func TestServiceThatFails(t *testing.T) {
	cfg, log := testSetup(t, time.Second)
	boom := errors.New("could not reach the database")

	err := supervise(context.Background(), func() {}, func(context.Context, config.Config, *slog.Logger) error {
		return boom
	}, cfg, log)

	if !errors.Is(err, boom) {
		t.Fatalf("got %v, want the service's own error back", err)
	}
}

func TestServiceStopsWhenAsked(t *testing.T) {
	cfg, log := testSetup(t, time.Second)
	ctx, cancel := context.WithCancel(context.Background())

	// Stands in for the signal arriving a moment after startup.
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	var sawStopSignal bool
	err := supervise(ctx, func() { sawStopSignal = true }, func(ctx context.Context, _ config.Config, _ *slog.Logger) error {
		<-ctx.Done()
		return nil
	}, cfg, log)

	if err != nil {
		t.Fatalf("got %v, want a clean stop", err)
	}
	if !sawStopSignal {
		t.Error("the default signal behaviour was never put back, so a second Ctrl-C would not kill the process")
	}
}

func TestContextCanceledIsNotTreatedAsAFailure(t *testing.T) {
	// A service that returns ctx.Err() after being cancelled did what it was told.
	// That must not look like a crash, or every clean shutdown exits non-zero.
	cfg, log := testSetup(t, time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := supervise(ctx, func() {}, func(ctx context.Context, _ config.Config, _ *slog.Logger) error {
		<-ctx.Done()
		return ctx.Err()
	}, cfg, log)

	if err != nil {
		t.Fatalf("got %v, want nil", err)
	}
}

func TestServiceThatWillNotStopInTime(t *testing.T) {
	// The safety net. A service that ignores the cancel must not hold the process
	// open forever: we give up, say so, and let whatever restarted us get on with it.
	cfg, log := testSetup(t, 50*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stubborn := make(chan struct{})
	t.Cleanup(func() { close(stubborn) })

	start := time.Now()
	err := supervise(ctx, func() {}, func(context.Context, config.Config, *slog.Logger) error {
		<-stubborn // never notices it was asked to stop
		return nil
	}, cfg, log)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("a service that ignored the cancel was reported as a clean shutdown")
	}
	if !strings.Contains(err.Error(), "did not finish draining") {
		t.Errorf("got %q, want it to say draining did not finish", err)
	}
	if elapsed > time.Second {
		t.Errorf("waited %s, expected to give up after the %s timeout", elapsed, cfg.ShutdownTimeout)
	}
}
