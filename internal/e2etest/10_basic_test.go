package e2etest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shoenig/test"
	launchcontrol "github.com/spikesdivzero/launch-control"
)

func TestSimpleDemo(t *testing.T) {
	eeTest(t, func(t *testing.T, c *launchcontrol.Controller, es eetState) {
		// The first service just runs until Stop is called.
		// This could be something like a net/http.Server, for instance.
		s1ch := make(chan struct{})
		c.Launch("s1", launchcontrol.Options{
			Run: func(ctx context.Context) error {
				<-s1ch
				return nil
			},
			Stop: func(ctx context.Context) error {
				close(s1ch)
				return nil
			},
		})

		// The second one is designed to fail after 10s,
		// demonstrating the way that any component failing shuts down everything.
		err := errors.New("something descriptive")
		s2StopCalled := false
		c.Launch("s2", launchcontrol.Options{
			Run: func(ctx context.Context) error {
				time.Sleep(10 * time.Second)
				return err
			},
			Stop: func(ctx context.Context) error {
				// This does actually get called, because the component started.
				// There may still be cleanup a user wants to preform!
				s2StopCalled = true
				return nil
			},
		})

		// The third one is just here to show that the failing service can be in the middle of the stack
		c.Launch("s3", launchcontrol.Options{
			Start: func(ctx context.Context) error { return nil },
			Stop:  func(ctx context.Context) error { return nil },
		})

		// The first non-nil error returned gets propegated
		test.ErrorIs(t, c.Wait(), err)
		test.True(t, s2StopCalled)
	})
}

// Stopping a new controller (that is, nothing has been launched in it) works like you'd expect it to.
func TestStopFromNew(t *testing.T) {
	eeTest(t, func(t *testing.T, c *launchcontrol.Controller, es eetState) {
		test.NoError(t, c.Err())

		c.RequestStop(nil)
		test.NoError(t, c.Err())

		// We can't launch new components inside of a stopped controller.
		c.Launch("foo", launchcontrol.Options{
			Run:  func(ctx context.Context) error { panic("shouldn't be called - already stopped") },
			Stop: func(ctx context.Context) error { panic("shouldn't be called - never starts") },
		})

		// Even if the controller's stopped, we still record errors sent to RequestStop
		err1 := errors.New("anything")
		c.RequestStop(err1)
		test.Eq(t, err1, c.Err())
	})
}
