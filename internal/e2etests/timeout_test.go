//go:build goexperiment.synctest

package e2etests

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/shoenig/test"
	"github.com/spikesdivzero/launch-control"
	"github.com/spikesdivzero/launch-control/internal/lcerrors"
)

func TestShutdownCallTimeout(t *testing.T) {
	synctest.Run(func() {
		ctrl := newController(t)
		ctrl.Launch("test",
			launch.WithRun(
				func(ctx context.Context) error {
					<-ctx.Done()
					return nil
				},
				func(ctx context.Context) error {
					time.Sleep(time.Minute)
					return nil
				}),
			launch.WithShutdownCallTimeout(5*time.Second))

		time.AfterFunc(time.Second, func() { ctrl.RequestStop(nil) })
		test.ErrorIs(t, ctrl.Wait(), lcerrors.ContextTimeoutError{Source: "Shutdown.CallTimeout"})
	})
}

func TestShutdownCompletionTimeout(t *testing.T) {
	synctest.Run(func() {
		ctrl := newController(t)
		ctrl.Launch("test",
			launch.WithRun(
				func(ctx context.Context) error {
					<-ctx.Done()
					return nil
				},
				func(ctx context.Context) error {
					time.Sleep(time.Minute)
					return nil
				}),
			launch.WithShutdownCompletionTimeout(5*time.Second))

		time.AfterFunc(time.Second, func() { ctrl.RequestStop(nil) })
		test.ErrorIs(t, ctrl.Wait(), lcerrors.ContextTimeoutError{Source: "Shutdown.CompletionTimeout"})
	})
}

func TestSSWStartCallTimeout(t *testing.T) {
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
		test.ErrorIs(t, ctrl.Wait(), lcerrors.ContextTimeoutError{Source: "StartStopWrapper.StartTimeout"})
	})
}

func TestSSWStopCallTimeout(t *testing.T) {
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

		test.ErrorIs(t, ctrl.Wait(), lcerrors.ContextTimeoutError{Source: "StartStopWrapper.StopTimeout"})
	})
}

func TestReadyCallTimeout(t *testing.T) {
	synctest.Run(func() {
		ctrl := newController(t)
		ctrl.Launch("test", withDummyStartStop(),
			launch.WithCheckReady(func(ctx context.Context) (bool, error) {
				time.Sleep(time.Minute)
				return true, nil
			}),
			launch.WithCheckReadyCallTimeout(2*time.Second))

		test.ErrorIs(t, ctrl.Wait(), lcerrors.ContextTimeoutError{Source: "CheckReady.CallTimeout"})
	})
}
