//go:build goexperiment.synctest

package e2etests

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/shoenig/test"
	"github.com/spikesdivzero/launch-control"
)

func TestShutdownTimeout(t *testing.T) {
	synctest.Run(func() {
		ctrl := newController(t)
		ctrl.Launch("test",
			launch.WithRun(
				func(ctx context.Context) error { return nil },
				func(ctx context.Context) error {
					time.Sleep(time.Minute)
					return nil
				}),
			launch.WithShutdownCallTimeout(5*time.Second))

		time.AfterFunc(time.Second, func() { ctrl.RequestStop(nil) })
		test.NoError(t, ctrl.Wait()) // TODO: should this report the timeout?
	})
}

func TestSSWStartTimeout(t *testing.T) {
	synctest.Run(func() {
		ctrl := newController(t)
		ctrl.Launch("test",
			launch.WithStartStop(
				func(ctx context.Context) error {
					time.Sleep(time.Minute)
					return nil
				},
				func(ctx context.Context) error { return nil }),
			launch.WithStartStopCallTimeouts(time.Second, time.Second))

		// The start timeout error should result in the system automatically shutting down
		// TOOD: should have a better error than "component test run exited: context deadline exceeded"
		test.ErrorIs(t, ctrl.Wait(), context.DeadlineExceeded)
	})
}

func TestSSWStopTimeout(t *testing.T) {
	synctest.Run(func() {
		ctrl := newController(t)
		ctrl.Launch("test",
			launch.WithStartStop(
				func(ctx context.Context) error { return nil },
				func(ctx context.Context) error {
					time.Sleep(time.Minute)
					return nil
				}),
			launch.WithStartStopCallTimeouts(2*time.Second, 2*time.Second))

		time.AfterFunc(time.Second, func() { ctrl.RequestStop(nil) })

		// TOOD: should have a better error than "component test run exited: context deadline exceeded"
		test.ErrorIs(t, ctrl.Wait(), context.DeadlineExceeded)
	})
}
