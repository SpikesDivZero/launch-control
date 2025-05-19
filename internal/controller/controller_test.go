//go:build goexperiment.synctest

package controller

import (
	"errors"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/shoenig/test"
	"github.com/shoenig/test/must"
	"github.com/spikesdivzero/launch-control/internal/lcerrors"
	"github.com/spikesdivzero/launch-control/internal/testutil"
)

func newTestingController(t *testing.T, initialState lifecycleState) *Controller {
	log := slog.New(slog.DiscardHandler)
	c := New(t.Context(), log)
	c.lifecycleState = initialState
	return c
}

func Test_newTestingController(t *testing.T) {
	for _, wantState := range []lifecycleState{lifecycleAlive, lifecycleDying} {
		c := newTestingController(t, wantState)
		test.Eq(t, wantState, c.lifecycleState)
	}
}

func TestNew(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	c := New(t.Context(), log)

	// We saved the args
	test.Eq(t, log, c.log)
	test.Eq(t, t.Context(), c.ctx)

	// We want to always have a buffer on requestLaunchCh -- both for our tests to use, and
	// to give ourselves a bigger safety margin to avoid a nasty deadlock.
	test.GreaterEq(t, 8, cap(c.requestLaunchCh))

	// We start in the new state, and the channel are both non-nil and open
	test.Eq(t, lifecycleNew, c.lifecycleState)

	test.NotNil(t, c.doneCh)
	testutil.ChanReadIsBlocked(t, c.doneCh)

	test.NotNil(t, c.requestStopCh)
	testutil.ChanReadIsBlocked(t, c.requestStopCh)

	test.NotNil(t, c.requestLaunchCh)
	testutil.ChanReadIsBlocked(t, c.requestLaunchCh)
}

func TestController_Launch(t *testing.T) {
	// The bulk of the launch logic is written in Controller.sendLaunchRequest, so is already tested elsewhere.
	// Here, we're mainly just looking to do a mini-test that Launch waits for the returned doneCh to be closed.
	synctest.Run(func() {
		c := newTestingController(t, lifecycleAlive)
		mc := &testutil.MockComponent{}

		launchDone := make(chan struct{})
		go func() {
			defer close(launchDone)
			c.Launch("test", mc)
		}()

		synctest.Wait()
		testutil.ChanReadIsBlocked(t, launchDone)

		// Check to see that it was connected
		test.True(t, mc.Recorder.Connect.Called)
		test.NotNil(t, mc.Recorder.Connect.NotifyOnExited) // Impl validated in TestController_Launch_notifyOnExit

		select {
		case req := <-c.requestLaunchCh:
			close(req.doneCh)
		case <-time.After(time.Second):
			t.Error("launchRequest not written?")
			return // FailNow not currently safe for use in synctest (go1.24 experimental)
		}

		synctest.Wait()
		testutil.ChanReadIsClosed(t, launchDone)
	})
}

func TestController_Launch_logError(t *testing.T) {
	t.Run("gets nil error", func(t *testing.T) {
		c := newTestingController(t, lifecycleAlive)
		mc := &testutil.MockComponent{}

		go func() {
			req := <-c.requestLaunchCh
			close(req.doneCh)
		}()
		c.Launch("test", mc)

		mc.Recorder.Connect.LogError("test", nil)
		test.NoError(t, c.Err())
	})

	t.Run("gets error", func(t *testing.T) {
		c := newTestingController(t, lifecycleAlive)
		mc := &testutil.MockComponent{}

		go func() {
			req := <-c.requestLaunchCh
			close(req.doneCh)
		}()
		c.Launch("test", mc)

		err := errors.New("anything")
		mc.Recorder.Connect.NotifyOnExited(err)
		test.ErrorIs(t, c.Err(), err)
	})
}

func TestController_Launch_notifyOnExit(t *testing.T) {
	t.Run("gets nil error", func(t *testing.T) {
		c := newTestingController(t, lifecycleAlive)
		mc := &testutil.MockComponent{}

		go func() {
			req := <-c.requestLaunchCh
			close(req.doneCh)
		}()
		c.Launch("test", mc)

		mc.Recorder.Connect.NotifyOnExited(nil)
		testutil.ChanReadIsClosed(t, c.requestStopCh) // called RequestShutdown
		test.NoError(t, c.Err())
	})

	t.Run("gets error", func(t *testing.T) {
		c := newTestingController(t, lifecycleAlive)
		mc := &testutil.MockComponent{}

		go func() {
			req := <-c.requestLaunchCh
			close(req.doneCh)
		}()
		c.Launch("test", mc)

		err := errors.New("anything")
		mc.Recorder.Connect.NotifyOnExited(err)
		testutil.ChanReadIsClosed(t, c.requestStopCh) // called RequestShutdown
		test.ErrorIs(t, c.Err(), err)
	})
}

func TestController_recordComponentError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		c := newTestingController(t, lifecycleAlive)
		c.recordComponentError("foo", "bar", nil)

		test.Len(t, 0, c.allErrors)
	})

	t.Run("gets error", func(t *testing.T) {
		c := newTestingController(t, lifecycleAlive)

		firstErr := errors.New("fancy")
		c.recordComponentError("foo", "bar", firstErr)
		must.Len(t, 1, c.allErrors)
		test.ErrorIs(t, c.allErrors[0], lcerrors.ComponentError{
			Name:  "foo",
			Stage: "bar",
			Err:   firstErr,
		})

		secondErr := errors.New("fancy")
		c.recordComponentError("jazz", "hands", secondErr)
		must.Len(t, 2, c.allErrors)
		test.ErrorIs(t, c.allErrors[1], lcerrors.ComponentError{
			Name:  "jazz",
			Stage: "hands",
			Err:   secondErr,
		})
	})
}

func TestController_sendLaunchRequest(t *testing.T) {
	t.Run("from New", func(t *testing.T) {
		synctest.Run(func() {
			// This test case is a bit more than a unit, since it'll launch the control loop.
			// Feels unavoidable, and I don't like it, but not much I can do about it.
			c := newTestingController(t, lifecycleNew)
			mc := &testutil.MockComponent{}

			doneCh := c.sendLaunchRequest("test", mc)
			must.NotNil(t, doneCh)

			// Control Loop processed this request.
			synctest.Wait()
			testutil.ChanReadIsClosed(t, doneCh)

			// Control loop should exit.
			c.RequestStop(nil)
			time.Sleep(dyingMonitorExitReportingGracePeriod)
			synctest.Wait()
			testutil.ChanReadIsClosed(t, c.doneCh)
		})
	})

	t.Run("from Alive", func(t *testing.T) {
		synctest.Run(func() {
			c := newTestingController(t, lifecycleAlive)
			mc := &testutil.MockComponent{}

			// This shouldn't launch the control loop, so our first channel state tests use that assumption
			doneCh := c.sendLaunchRequest("test", mc)
			test.False(t, mc.Recorder.Start.Called)
			testutil.ChanReadIsBlocked(t, doneCh)
			testutil.ChanReadIsOk(t, c.requestLaunchCh, launchRequest{"test", mc, doneCh})

			// Verify that the control loop wasn't launched
			dummyReq := launchRequest{"test123", mc, make(chan struct{})}
			c.requestLaunchCh <- dummyReq
			synctest.Wait() // If the loop is running, this will let it eat the request
			testutil.ChanReadIsOk(t, c.requestLaunchCh, dummyReq)
		})
	})

	t.Run("from Dead/Dying", func(t *testing.T) {
		for _, state := range []lifecycleState{lifecycleDying, lifecycleDead} {
			synctest.Run(func() {
				c := newTestingController(t, state)
				mc := &testutil.MockComponent{}

				// This shouldn't launch the control loop, so our first channel state tests use that assumption
				doneCh := c.sendLaunchRequest("test", mc)
				testutil.ChanReadIsClosed(t, doneCh)             // should be pre-closed
				testutil.ChanReadIsBlocked(t, c.requestLaunchCh) // request shouldn't have been written
				test.False(t, mc.Recorder.Start.Called)

				// Verify that the control loop wasn't launched
				dummyReq := launchRequest{"test123", mc, make(chan struct{})}
				c.requestLaunchCh <- dummyReq
				synctest.Wait() // If the loop is running, this will let it eat the request
				testutil.ChanReadIsOk(t, c.requestLaunchCh, dummyReq)
			})
		}
	})
}

func TestController_RequestStop(t *testing.T) {
	t.Run("from Alive", func(t *testing.T) {
		c := newTestingController(t, lifecycleAlive)
		check := func(wantErrIs error) {
			test.ErrorIs(t, c.Err(), wantErrIs)

			test.Eq(t, lifecycleAlive, c.lifecycleState)
			testutil.ChanReadIsClosed(t, c.requestStopCh)

			testutil.ChanReadIsBlocked(t, c.doneCh)
			testutil.ChanReadIsBlocked(t, c.requestLaunchCh)
		}

		// {nil, err1, err2, nil}
		c.RequestStop(nil)
		check(nil)

		testErr := errors.New("testy")
		c.RequestStop(testErr)
		check(testErr)

		c.RequestStop(errors.New("something else"))
		check(testErr)

		c.RequestStop(nil)
		check(testErr)
	})

	t.Run("New-Dead skip transition", func(t *testing.T) {
		c := newTestingController(t, lifecycleNew)
		check := func(wantErrIs error) {
			test.ErrorIs(t, c.Err(), wantErrIs)

			test.Eq(t, lifecycleDead, c.lifecycleState)
			testutil.ChanReadIsClosed(t, c.requestStopCh)

			testutil.ChanReadIsClosed(t, c.doneCh)
			testutil.ChanReadIsClosed(t, c.requestLaunchCh)
		}

		// {nil, err1, err2, nil}
		c.RequestStop(nil)
		check(nil)

		testErr := errors.New("testy")
		c.RequestStop(testErr)
		check(testErr)

		c.RequestStop(errors.New("something else"))
		check(testErr)

		c.RequestStop(nil)
		check(testErr)
	})
}

func TestController_Wait(t *testing.T) {
	synctest.Run(func() {
		c := newTestingController(t, lifecycleNew)

		resultCh := make(chan error)
		go func() {
			defer close(resultCh)
			resultCh <- c.Wait()
		}()

		synctest.Wait()
		testutil.ChanReadIsBlocked(t, resultCh)

		testErr := errors.New("hello")
		c.allErrors = append(c.allErrors, testErr)
		close(c.doneCh)

		synctest.Wait()
		testutil.ChanReadIsOk(t, resultCh, testErr)
	})
}

func TestController_Err(t *testing.T) {
	c := newTestingController(t, lifecycleNew)
	test.Nil(t, c.Err())

	testErr := errors.New("testy")
	c.allErrors = append(c.allErrors, testErr, errors.New("something else"))
	test.ErrorIs(t, c.Err(), testErr)
}

func TestController_AllErrors(t *testing.T) {
	c := newTestingController(t, lifecycleNew)
	test.Nil(t, c.AllErrors())

	c.allErrors = append(c.allErrors, errors.New("first"), errors.New("second"))

	got := c.AllErrors()
	test.SliceEqOp(t, c.allErrors, got)

	// We return a copy so the user can't modify our internal state.
	got[0] = errors.New("different")
	test.NotEqOp(t, c.allErrors[0], got[0])
}
