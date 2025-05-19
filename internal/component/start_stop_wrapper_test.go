//go:build goexperiment.synctest

package component

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"testing/synctest"
	"time"

	"github.com/shoenig/test"
	"github.com/shoenig/test/must"
	"github.com/spikesdivzero/launch-control/internal/testutil"
)

func newStartStopWrapper(*testing.T) StartStopWrapper {
	return StartStopWrapper{
		ImplStart: func(context.Context) error { panic("ImplStart called, but not defined in test") },
		ImplStop:  func(context.Context) error { panic("ImplStop called, but not defined in test") },

		StartTimeout: NoTimeout,
		StopTimeout:  NoTimeout,
	}
}

func TestStartStopWrapper_Run(t *testing.T) {
	// Conviently, testutil.MockComponent has the same basic signatures as our own Start/Stop, so we'll (ab)use that.

	t.Run("prevent double call", func(t *testing.T) {
		defer testutil.WantPanic(t, "internal: StartStopWrapper Run called twice")
		ssw := newStartStopWrapper(t)
		ssw.requestStopCh = make(chan struct{})
		_ = ssw.Run(t.Context())
	})

	// Well, mostly happy. When start returns nil, both should be called, and it should return the final error from Stop.
	t.Run("happy", func(t *testing.T) {
		synctest.Run(func() {

			mc := &testutil.MockComponent{}
			mc.ShutdownOptions.Err = errors.New("fancy")

			ssw := newStartStopWrapper(t)
			ssw.ImplStart = mc.Start
			ssw.ImplStop = mc.Shutdown

			resultCh := make(chan error)
			go func() {
				defer close(resultCh)
				resultCh <- ssw.Run(t.Context())
			}()

			synctest.Wait()
			testutil.ChanReadIsBlocked(t, resultCh)

			must.NotNil(t, ssw.requestStopCh)
			close(ssw.requestStopCh)

			synctest.Wait()
			err := <-resultCh
			test.ErrorIs(t, err, mc.ShutdownOptions.Err)
		})
	})

	t.Run("start returns error", func(t *testing.T) {
		mc := &testutil.MockComponent{}
		mc.StartOptions.Err = errors.New("testy")

		ssw := newStartStopWrapper(t)
		ssw.ImplStart = mc.Start

		err := ssw.Run(t.Context())
		test.True(t, mc.Recorder.Start.Called)
		test.ErrorIs(t, err, mc.StartOptions.Err)
	})
}

func TestStartStopWrapper_Shutdown(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		ssw := newStartStopWrapper(t)
		ssw.requestStopCh = make(chan struct{})
		err := ssw.Shutdown(t.Context())
		test.Nil(t, err)
		testutil.ChanReadIsClosed(t, ssw.requestStopCh)
	})

	t.Run("chan starts nil", func(t *testing.T) {
		ssw := newStartStopWrapper(t)
		err := ssw.Shutdown(t.Context())
		test.Nil(t, err)
		test.NotNil(t, ssw.requestStopCh)
		testutil.ChanReadIsClosed(t, ssw.requestStopCh)
	})
}

func TestStartStopWrapper_doCall(t *testing.T) {
	ssw := newStartStopWrapper(t)

	// Capture the callErr
	err := ssw.doCall(t.Context(), "in-test", time.Second, func(ctx context.Context) error {
		runtime.Goexit() // called within the AsyncCall coroutine
		panic("unreachable")
	})
	test.ErrorIs(t, err, errPrematureChannelClose)

	// And the basic error
	testErr := errors.New("goose")
	err = ssw.doCall(t.Context(), "in-test", time.Second, func(ctx context.Context) error {
		return testErr
	})
	test.ErrorIs(t, err, testErr)
}
