package controller

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/shoenig/test"
	"github.com/spikesdivzero/launch-control/internal/lcerrors"
	"github.com/spikesdivzero/launch-control/internal/testutil"
)

func TestController_controlLoop_Dying(t *testing.T) {
	t.Run("mismatch state", func(t *testing.T) {
		defer testutil.WantPanic(t, "internal: controlLoop_Dying clAssertState, state Alive, expected Dying")
		c := newTestingController(t, lifecycleAlive)
		c.controlLoop_Dying()
	})

	t.Run("happy", func(t *testing.T) {
		c := newTestingController(t, lifecycleDying)

		// Fill up the requests with things that shouldn't get executed
		nocallMc := &testutil.MockComponent{}
		test.GreaterEq(t, 8, cap(c.requestLaunchCh))
		reqs := make([]launchRequest, cap(c.requestLaunchCh))
		for i := range reqs {
			reqs[i] = launchRequest{"test", nocallMc, make(chan struct{})}
			c.requestLaunchCh <- reqs[i]
		}

		// And create some components to be shutdown
		gotShutdownOrder := []string{}
		componentNames := []string{}
		for i := range 5 {
			name := fmt.Sprintf("comp-%v", i)
			componentNames = append(componentNames, name)

			mc := &testutil.MockComponent{}
			mc.ShutdownOptions.Hook = func() { gotShutdownOrder = append(gotShutdownOrder, name) }

			c.components = append(c.components, ownedComponent{name, mc})
		}

		// Now let it run
		c.controlLoop_Dying()

		// Check our discards
		testutil.ChanReadIsClosed(t, c.requestLaunchCh) // everything abandoned
		for _, req := range reqs {
			testutil.ChanReadIsClosed(t, req.doneCh) // all unblocked
		}

		// And check our shutdown order
		slices.Reverse(componentNames)
		test.Eq(t, componentNames, gotShutdownOrder)
	})
}

func TestController_clDyingDoShutdown(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		c := newTestingController(t, lifecycleDying)
		mc := &testutil.MockComponent{}
		c.clDyingDoShutdown(ownedComponent{"test-comp", mc})
		test.True(t, mc.Recorder.Shutdown.Called)
	})

	t.Run("returns err", func(t *testing.T) {
		c := newTestingController(t, lifecycleDying)
		mc := &testutil.MockComponent{}
		mc.ShutdownOptions.Err = errors.New("test error")
		c.clDyingDoShutdown(ownedComponent{"test-comp", mc})
		test.True(t, mc.Recorder.Shutdown.Called)
		test.ErrorIs(t, c.Err(), lcerrors.ComponentError{
			Name:  "test-comp",
			Stage: "shutdown",
			Err:   mc.ShutdownOptions.Err,
		})
	})
}
