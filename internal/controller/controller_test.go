package controller

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/shoenig/test"
	"github.com/shoenig/test/must"
	"github.com/spikesdivzero/launch-control/internal/component"
	"github.com/spikesdivzero/launch-control/internal/testutil"
)

func init() {
	// Provides "coverage" for stringer error branch
	_ = runState(-1).String()

	// Type assertions to ensure the interface is fulfilled.
	_ = Component(&component.Component{})
	_ = Component(&component.StubComponent{})
}

func TestController_recordError(t *testing.T) {
	for _, tt := range []struct {
		name        string
		useErr      bool // false => use nil
		acquireLock bool
	}{
		{"nil-nolock", false, false},
		{"nil-lock", false, true},
		{"err-nolock", true, false},
		{"err-lock", true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err1, err2 := errors.New("err1"), errors.New("err2")

			c := NewController(t.Context())
			c.errs = []error{err1}

			wantErrs := c.errs[:]
			var errArg error
			if tt.useErr {
				errArg = err2
				wantErrs = append(wantErrs, err2)
			}

			c.recordError(errArg, tt.acquireLock)

			test.SliceEqOp(t, wantErrs, c.errs)
		})
	}
}

func TestController_Launch(t *testing.T) {
	// Don't launch in Dead/Dying states. These also shouldn't start the worker.
	for _, state := range []runState{runStateDying, runStateDead} {
		t.Run(fmt.Sprint("state", state), func(t *testing.T) {
			c := NewController(t.Context())
			c.state = state

			// We don't define ImplRun. If the worker is started and attempts to launch this component, it panics.
			comp := &component.StubComponent{}

			c.Launch("testing", comp)   // Should not block
			test.SliceEmpty(t, c.comps) // Nothing gets registered

		})
	}

	t.Run("state Alive", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			c := NewController(t.Context())
			c.state = runStateAlive // Don't launch the worker a second time

			// We don't define ImplRun. If the worker is started and attempts to launch this component, it panics.
			comp := &component.StubComponent{}

			launchReturned := make(chan struct{})
			go func() {
				defer close(launchReturned)
				c.Launch("testing", comp)
			}()
			synctest.Wait()

			testutil.ChanReadIsBlocked(t, launchReturned)

			select {
			case req := <-c.requestLaunchCh:
				close(req.DoneCh)
			default:
				t.Fatal("Launch didn't write request to channel, or worker started")
			}

			synctest.Wait()

			testutil.ChanReadIsClosed(t, launchReturned)
		})
	})

	// This one ends up being a bit of an end-to-end test since we start the worker.
	// Our ImplRun returns an error, so the worker should shut down.
	// We use a synctest bubble to confirm the shutdown occurs. If it doesn't exit, synctest will panic.
	t.Run("state New", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			c := NewController(t.Context())

			comp := &component.StubComponent{
				ImplRegister: func(ctx context.Context, l *slog.Logger, cc component.ControllerCallbacks) {},
				ImplStart: func(ctx context.Context) {
					c.RequestStop(errors.New("shutdown1"))
				},
				ImplStop: func() {},
			}

			c.Launch("testing", comp)
			time.Sleep(6 * time.Second) // worker_discardLaunchRequests runs for 5s
		})
	})
}

func TestController_registerComponent(t *testing.T) {
	c := NewController(t.Context())

	log := slog.New(slog.DiscardHandler)
	c.SetLogger(log)

	var registerCalled bool
	comp := &component.StubComponent{
		ImplRegister: func(ctx context.Context, l *slog.Logger, cc component.ControllerCallbacks) {
			registerCalled = true
			test.EqOp(t, t.Context(), ctx)
			test.EqOp(t, log, l)
			test.NotNil(t, cc.ComponentError)
			test.NotNil(t, cc.RequestStop)
		},
	}

	c.registerComponent("testing", comp)
	test.True(t, registerCalled)
}

func TestController_makeControllerCallbacksFor(t *testing.T) {
	c := NewController(t.Context())
	comp := &component.StubComponent{}
	cb := c.makeControllerCallbacksFor("testing", comp)

	err1 := errors.New("err1")

	must.NotNil(t, cb.ComponentError)
	c.errs = nil
	cb.ComponentError(err1)
	cb.ComponentError(nil)
	cb.ComponentError(err1)
	test.SliceEqOp(t, c.errs, []error{err1, err1})

	testutil.ChanReadIsBlocked(t, c.requestStopCh)

	// RequestStop is a thin wrapper around c.RequestStop, so we won't test it in detail here.
	must.NotNil(t, cb.RequestStop)
	c.errs = nil
	cb.RequestStop(err1)
	cb.RequestStop(nil)
	cb.RequestStop(err1)
	test.SliceEqOp(t, c.errs, []error{err1, err1})

	testutil.ChanReadIsClosed(t, c.requestStopCh)
}

func TestController_RequestStop(t *testing.T) {
	t.Run("from New", func(t *testing.T) {
		c := NewController(t.Context())

		c.RequestStop(nil)
		test.Eq(t, runStateDead, c.state)
		testutil.ChanReadIsClosed(t, c.requestLaunchCh)
		testutil.ChanReadIsClosed(t, c.requestStopCh)
		testutil.ChanReadIsClosed(t, c.deadCh)

		// Cancelling multiple times should be fine.
		err := errors.New("test err")
		c.RequestStop(err)
		c.RequestStop(nil)
		test.ErrorIs(t, c.Err(), err)
	})

	t.Run("from Alive", func(t *testing.T) {
		c := NewController(t.Context())
		c.state = runStateAlive

		c.RequestStop(nil)
		test.Eq(t, runStateAlive, c.state)
		testutil.ChanReadIsBlocked(t, c.requestLaunchCh)
		testutil.ChanReadIsClosed(t, c.requestStopCh)
		testutil.ChanReadIsBlocked(t, c.deadCh)

		// Cancelling multiple times should be fine.
		err := errors.New("test err")
		c.RequestStop(err)
		c.RequestStop(nil)
		test.ErrorIs(t, c.Err(), err)
	})
}

func TestController_Wait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		c := NewController(t.Context())

		waitReturned := make(chan struct{})
		go func() {
			defer close(waitReturned)
			c.Wait()
		}()
		synctest.Wait()

		testutil.ChanReadIsBlocked(t, waitReturned, test.Sprint("Wait has not finished"))

		close(c.deadCh)
		synctest.Wait()

		testutil.ChanReadIsClosed(t, waitReturned, test.Sprint("Wait finished"))
	})
}

func TestController_Err(t *testing.T) {
	c := NewController(t.Context())
	test.NoError(t, c.Err())

	c.errs = []error{errors.New("err1"), errors.New("err2")}
	test.EqOp(t, c.errs[0], c.Err())
}

func TestController_AllErrors(t *testing.T) {
	c := NewController(t.Context())
	test.NoError(t, c.Err())

	c.errs = []error{errors.New("err1"), errors.New("err2")}
	ret := c.AllErrors()
	test.SliceEqOp(t, c.errs, ret)

	// ret should be a copy of c.errs
	ret[0] = errors.New("err3")
	test.NotEqOp(t, c.errs[0], ret[0])
}

func TestController_SetLogger(t *testing.T) {
	log := slog.New(slog.DiscardHandler)

	t.Run("ok New", func(t *testing.T) {
		c := NewController(t.Context())
		c.SetLogger(log)
	})

	for _, state := range []runState{runStateAlive, runStateDying, runStateDead} {
		name := fmt.Sprint("fail", state)
		t.Run(name, func(t *testing.T) {
			defer testutil.WantPanic(t, "Controller.SetLogger is forbidden after the first Launch/RequestStop call")
			c := NewController(t.Context())
			c.state = state
			c.SetLogger(log)
		})
	}
}
