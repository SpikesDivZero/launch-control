//go:build goexperiment.synctest

package controller

import (
	"errors"
	"testing"
	"testing/synctest"

	"github.com/shoenig/test"
	"github.com/spikesdivzero/launch-control/internal/testutil"
)

func TestController_controlLoop_Alive(t *testing.T) {
	t.Run("checks state", func(t *testing.T) {
		defer testutil.WantPanic(t, "internal: controlLoop_Alive clAssertState, state New, expected Alive")
		c := newTestingController(t, lifecycleNew)
		c.controlLoop_Alive()
	})

	t.Run("main loop", func(t *testing.T) {
		synctest.Run(func() {
			c := newTestingController(t, lifecycleAlive)

			clExited := make(chan struct{})
			go func() {
				defer close(clExited)
				c.controlLoop_Alive()
			}()

			synctest.Wait()
			testutil.ChanReadIsBlocked(t, clExited)

			for range 8 {
				mc := &testutil.MockComponent{}
				doneCh := make(chan struct{})
				c.requestLaunchCh <- launchRequest{"test", mc, doneCh}

				synctest.Wait()
				testutil.ChanReadIsClosed(t, doneCh)    // finished
				testutil.ChanReadIsBlocked(t, clExited) // still alive
			}

			close(c.requestStopCh)
			synctest.Wait()
			testutil.ChanReadIsClosed(t, clExited) // responded
		})
	})
}

func TestController_clAliveDoLaunch(t *testing.T) {
	makeReq := func() (*testutil.MockComponent, launchRequest) {
		mc := &testutil.MockComponent{}
		return mc, launchRequest{"test", mc, make(chan struct{})}
	}

	t.Run("discard when stop requested", func(t *testing.T) {
		c := newTestingController(t, lifecycleAlive)
		mc, req := makeReq()

		close(c.requestStopCh)

		c.clAliveDoLaunch(req)
		testutil.ChanReadIsClosed(t, req.doneCh)
		test.False(t, mc.Recorder.Start.Called)
		test.Len(t, 0, c.components)
	})

	t.Run("happy", func(t *testing.T) {
		c := newTestingController(t, lifecycleAlive)
		mc, req := makeReq()

		c.clAliveDoLaunch(req)
		testutil.ChanReadIsClosed(t, req.doneCh)
		test.True(t, mc.Recorder.Start.Called)
		test.True(t, mc.Recorder.WaitReady.Called)
		test.Len(t, 1, c.components)
		test.Eq(t, mc, c.components[0].(*testutil.MockComponent))

		test.Eq(t, c.requestStopCh, mc.Recorder.WaitReady.AbortLoopCh)

		testutil.ChanReadIsBlocked(t, c.requestStopCh) // no stop requested
	})

	t.Run("start returns error", func(t *testing.T) {
		c := newTestingController(t, lifecycleAlive)
		mc, req := makeReq()

		testErr := errors.New("boop")
		mc.StartOptions.Err = testErr

		c.clAliveDoLaunch(req)
		testutil.ChanReadIsClosed(t, req.doneCh)
		test.True(t, mc.Recorder.Start.Called)
		test.False(t, mc.Recorder.WaitReady.Called)
		test.Len(t, 1, c.components)
		test.Eq(t, mc, c.components[0].(*testutil.MockComponent))

		testutil.ChanReadIsClosed(t, c.requestStopCh)
		test.ErrorIs(t, c.firstError, testErr)
	})

	t.Run("wait-ready returns error", func(t *testing.T) {
		c := newTestingController(t, lifecycleAlive)
		mc, req := makeReq()

		testErr := errors.New("shoes")
		mc.WaitReadyOptions.Err = testErr

		c.clAliveDoLaunch(req)
		testutil.ChanReadIsClosed(t, req.doneCh)
		test.True(t, mc.Recorder.Start.Called)
		test.True(t, mc.Recorder.WaitReady.Called)
		test.Len(t, 1, c.components)
		test.Eq(t, mc, c.components[0].(*testutil.MockComponent))

		testutil.ChanReadIsClosed(t, c.requestStopCh)
		test.ErrorIs(t, c.firstError, testErr)
	})
}
