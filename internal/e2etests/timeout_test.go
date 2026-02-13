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
	synctest.Test(t, func(t *testing.T) {
		ctrl := newController(t)
		ctrl.Launch126("test", launch.Options126{
			Run: func(ctx context.Context) error {
				<-ctx.Done()
				return nil
			},
			Shutdown: func(ctx context.Context) error {
				time.Sleep(time.Minute)
				return nil
			},
			ShutdownCallTimeout: new(5 * time.Second),
		})

		time.AfterFunc(time.Second, func() { ctrl.RequestStop(nil) })
		test.ErrorIs(t, ctrl.Wait(), lcerrors.ContextTimeoutError{Source: "Shutdown.CallTimeout"})
	})
}

func TestShutdownCompletionTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctrl := newController(t)
		ctrl.Launch126("test", launch.Options126{
			Run: func(ctx context.Context) error {
				<-ctx.Done()
				return nil
			},
			Shutdown: func(ctx context.Context) error {
				time.Sleep(time.Minute)
				return nil
			},
			ShutdownCompletionTimeout: new(5 * time.Second),
		})

		time.AfterFunc(time.Second, func() { ctrl.RequestStop(nil) })
		test.ErrorIs(t, ctrl.Wait(), lcerrors.ContextTimeoutError{Source: "Shutdown.CompletionTimeout"})

		// HACK(go1.25 upgrade): some of our test coroutines run longer than our main test, causing a panic.
		// Sleep at end fixes this, for now. I should redo this later on to be smarter.
		time.Sleep(5 * time.Minute)
	})
}

func TestSSWStartCallTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctrl := newController(t)
		ctrl.Launch126("test", launch.Options126{
			Start: func(ctx context.Context) error {
				time.Sleep(time.Minute)
				return nil
			},
			Stop:             func(ctx context.Context) error { return nil },
			StartCallTimeout: new(time.Second),
			StopCallTimeout:  new(time.Second),
		})

		// The start timeout error should result in the system automatically shutting down
		test.ErrorIs(t, ctrl.Wait(), lcerrors.ContextTimeoutError{Source: "StartStopWrapper.StartTimeout"})

		// HACK(go1.25 upgrade): some of our test coroutines run longer than our main test, causing a panic.
		// Sleep at end fixes this, for now. I should redo this later on to be smarter.
		time.Sleep(5 * time.Minute)
	})
}

func TestSSWStopCallTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctrl := newController(t)
		ctrl.Launch126("test", launch.Options126{
			Start: func(ctx context.Context) error { return nil },
			Stop: func(ctx context.Context) error {
				time.Sleep(time.Minute)
				return nil
			},
			StartCallTimeout: new(2 * time.Second),
			StopCallTimeout:  new(2 * time.Second),
		})

		time.AfterFunc(time.Second, func() { ctrl.RequestStop(nil) })

		test.ErrorIs(t, ctrl.Wait(), lcerrors.ContextTimeoutError{Source: "StartStopWrapper.StopTimeout"})

		// HACK(go1.25 upgrade): some of our test coroutines run longer than our main test, causing a panic.
		// Sleep at end fixes this, for now. I should redo this later on to be smarter.
		time.Sleep(5 * time.Minute)
	})
}

func TestReadyCallTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctrl := newController(t)
		ctrl.Launch126("test",
			withDummyStartStop(),
			launch.Options126{
				CheckReady: func(ctx context.Context) (bool, error) {
					time.Sleep(time.Minute)
					return true, nil
				},
				CheckReadyCallTimeout: new(2 * time.Second),
			})

		test.ErrorIs(t, ctrl.Wait(), lcerrors.ContextTimeoutError{Source: "CheckReady.CallTimeout"})

		// HACK(go1.25 upgrade): some of our test coroutines run longer than our main test, causing a panic.
		// Sleep at end fixes this, for now. I should redo this later on to be smarter.
		time.Sleep(5 * time.Minute)
	})
}
