package controller

import (
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"testing/synctest"

	"github.com/shoenig/test"
	"github.com/spikesdivzero/launch-control/internal/testutil"
)

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
