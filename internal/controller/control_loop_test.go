//go:build goexperiment.synctest

package controller

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/shoenig/test"
	"github.com/spikesdivzero/launch-control/internal/testutil"
)

// Silly bit to provide "coverage" for a stringer branch.
func init() { _ = lifecycleState(-1).String() }

func TestController_controlLoop(t *testing.T) {
	synctest.Run(func() {
		// This ends up being another broader test instead of a unit test, but I think that's OK with me, since
		// I want to ensure it runs normally.
		c := newTestingController(t, lifecycleAlive)

		innerDone := make(chan struct{})
		go func() {
			defer close(innerDone)
			c.controlLoop()
		}()

		synctest.Wait()
		testutil.ChanReadIsBlocked(t, innerDone)

		// The internal state changes are assessed via panic calls in the Alive and Dying funcs

		c.RequestStop(nil)
		time.Sleep(dyingMonitorExitReportingGracePeriod)
		synctest.Wait()

		testutil.ChanReadIsClosed(t, innerDone)

		test.Eq(t, lifecycleDead, c.lifecycleState)
		testutil.ChanReadIsClosed(t, c.doneCh)
	})
}

func TestController_clAssertState(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		c := newTestingController(t, lifecycleAlive)
		c.clAssertState("happy1", lifecycleAlive)

		c.lifecycleState = lifecycleDying
		c.clAssertState("happy2", lifecycleDying)
	})

	t.Run("mismatch", func(t *testing.T) {
		defer testutil.WantPanic(t, "internal: mismatch1 clAssertState, state New, expected Alive")
		c := newTestingController(t, lifecycleNew)
		c.clAssertState("mismatch1", lifecycleAlive)
	})
}

func TestController_clSetState(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		c := newTestingController(t, lifecycleNew)
		c.clSetState(lifecycleNew, lifecycleAlive)
		c.clSetState(lifecycleAlive, lifecycleDying)
		c.clSetState(lifecycleDying, lifecycleDead)

		// Odd, but it shouldn't be an panic
		c.clSetState(lifecycleDead, lifecycleDead)
	})

	t.Run("bad from", func(t *testing.T) {
		defer testutil.WantPanic(t, "internal: SetState from state New, expected Alive")
		c := newTestingController(t, lifecycleNew)
		c.clSetState(lifecycleAlive, lifecycleDying)
	})
}
