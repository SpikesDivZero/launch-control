//go:build goexperiment.synctest

package e2etests

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/shoenig/test"
	"github.com/spikesdivzero/launch-control"
)

// For many of these tests, I'm going to be wrapping them in synctest.Run mainly for the benefit of deadlock detection.
// Also, if any timings concerns do come into play, we won't end up having tests run for hours.

func newController(t *testing.T) launch.Controller {
	return launch.NewController(t.Context(), slog.New(slog.DiscardHandler))
}

// We shouldn't panic if Wait is called with no prior calls to Launch.
func TestStopWaitWithNoComponents(t *testing.T) {
	synctest.Run(func() {
		ctrl := newController(t)

		err := errors.New("hello")
		time.AfterFunc(time.Second, func() { ctrl.RequestStop(err) })

		test.ErrorIs(t, ctrl.Wait(), err)
		test.ErrorIs(t, ctrl.Err(), err)
	})
}

func TestRunExitingWithErrorCausesShutdown(t *testing.T) {
	synctest.Run(func() {
		ctrl := newController(t)

		err := errors.New("boop")

		ctrl.Launch("one", launch.WithStartStop(
			func(ctx context.Context) error { return nil },
			func(ctx context.Context) error { return nil }))
		ctrl.Launch("two", launch.WithRun(
			func(ctx context.Context) error {
				time.Sleep(time.Second)
				return err
			},
			func(ctx context.Context) error { return nil }))
		ctrl.Launch("three", launch.WithStartStop(
			func(ctx context.Context) error { return nil },
			func(ctx context.Context) error { return nil }))

		test.ErrorIs(t, ctrl.Wait(), err)
	})
}

// Similar, but returns no error. The mere fact that a component exited is enough to cause a shutdown
func TestRunExitingWithNoErrorCausesShutdown(t *testing.T) {
	synctest.Run(func() {
		ctrl := newController(t)

		ctrl.Launch("one", launch.WithStartStop(
			func(ctx context.Context) error { return nil },
			func(ctx context.Context) error { return nil }))
		ctrl.Launch("two", launch.WithRun(
			func(ctx context.Context) error {
				time.Sleep(time.Second)
				return nil
			},
			func(ctx context.Context) error { return nil }))
		ctrl.Launch("three", launch.WithStartStop(
			func(ctx context.Context) error { return nil },
			func(ctx context.Context) error { return nil }))

		test.ErrorIs(t, ctrl.Wait(), nil)
	})
}
