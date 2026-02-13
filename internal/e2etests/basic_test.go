package e2etests

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/shoenig/test"
	"github.com/spikesdivzero/launch-control"
)

// We shouldn't panic if Wait is called with no prior calls to Launch.
func TestStopWaitWithNoComponents(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctrl := newController(t)

		err := errors.New("hello")
		time.AfterFunc(time.Second, func() { ctrl.RequestStop(err) })

		test.ErrorIs(t, ctrl.Wait(), err)
		test.ErrorIs(t, ctrl.Err(), err)
	})
}

func TestRunExitingWithErrorCausesShutdown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctrl := newController(t)

		err := errors.New("boop")

		ctrl.Launch126("one", launch.Options126{
			Start: func(ctx context.Context) error { return nil },
			Stop:  func(ctx context.Context) error { return nil }})
		ctrl.Launch126("two", launch.Options126{
			Start: func(ctx context.Context) error {
				time.Sleep(time.Second)
				return err
			},
			Stop: func(ctx context.Context) error { return nil }})
		ctrl.Launch126("three", launch.Options126{
			Start: func(ctx context.Context) error { return nil },
			Stop:  func(ctx context.Context) error { return nil }})

		test.ErrorIs(t, ctrl.Wait(), err)
	})
}

// Similar, but returns no error. The mere fact that a component exited is enough to cause a shutdown
func TestRunExitingWithNoErrorCausesShutdown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctrl := newController(t)

		ctrl.Launch126("one", launch.Options126{
			Start: func(ctx context.Context) error { return nil },
			Stop:  func(ctx context.Context) error { return nil }})
		ctrl.Launch126("two", launch.Options126{
			Run: func(ctx context.Context) error {
				time.Sleep(time.Second)
				return nil
			},
			Shutdown: func(ctx context.Context) error { return nil }})
		ctrl.Launch126("three", launch.Options126{
			Start: func(ctx context.Context) error { return nil },
			Stop:  func(ctx context.Context) error { return nil }})

		test.ErrorIs(t, ctrl.Wait(), nil)
	})
}
