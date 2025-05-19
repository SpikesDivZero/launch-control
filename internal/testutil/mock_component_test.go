//go:build goexperiment.synctest

package testutil

import (
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/shoenig/test"
)

func TestMockComponent_ConnectController(t *testing.T) {
	var testNotifyGot error
	testNotify := func(err error) { testNotifyGot = err }

	var testLogErrorGot struct {
		stage string
		err   error
	}
	testLogError := func(stage string, err error) {
		testLogErrorGot.stage = stage
		testLogErrorGot.err = err
	}

	mc := &MockComponent{}
	mc.ConnectController(testLogError, testNotify)

	testErr := errors.New("boop")
	mc.Recorder.Connect.NotifyOnExited(testErr)
	test.ErrorIs(t, testNotifyGot, testErr)

	mc.Recorder.Connect.LogError("in-test", testErr)
	test.Eq(t, testLogErrorGot.stage, "in-test")
	test.ErrorIs(t, testLogErrorGot.err, testErr)
}

func TestMockComponent_Start(t *testing.T) {
	synctest.Run(func() {
		mc := &MockComponent{}
		test.False(t, mc.Recorder.Start.Called)

		commonChecks := func(wantD time.Duration, wantErrIs error) {
			mc.Recorder.Start.Called = false
			mc.Recorder.Start.Ctx = nil

			t0 := time.Now()
			err := mc.Start(t.Context())
			d := time.Since(t0)

			test.Eq(t, wantD, d)
			test.ErrorIs(t, err, wantErrIs)
			test.True(t, mc.Recorder.Start.Called)
			test.Eq(t, t.Context(), mc.Recorder.Start.Ctx)
		}

		// Should work normally with no args
		commonChecks(0, nil)

		// Our different controls should work
		testErr := errors.New("ooga")
		hookCalled := false
		mc.StartOptions.Err = testErr
		mc.StartOptions.Sleep = 3 * time.Second
		mc.StartOptions.Hook = func() { hookCalled = true }

		commonChecks(3*time.Second, testErr)
		test.True(t, hookCalled)
	})
}

func TestMockComponent_Shutdown(t *testing.T) {
	synctest.Run(func() {
		mc := &MockComponent{}
		test.False(t, mc.Recorder.Shutdown.Called)

		commonChecks := func(wantD time.Duration, wantErrIs error) {
			mc.Recorder.Shutdown.Called = false
			mc.Recorder.Shutdown.Ctx = nil

			t0 := time.Now()
			err := mc.Shutdown(t.Context())
			d := time.Since(t0)

			test.Eq(t, wantD, d)
			test.ErrorIs(t, err, wantErrIs)
			test.True(t, mc.Recorder.Shutdown.Called)
			test.Eq(t, t.Context(), mc.Recorder.Shutdown.Ctx)
		}

		// Should work normally with no args
		commonChecks(0, nil)

		// Our different controls should work
		testErr := errors.New("aiph")
		hookCalled := false
		mc.ShutdownOptions.Err = testErr
		mc.ShutdownOptions.Sleep = 3 * time.Second
		mc.ShutdownOptions.Hook = func() { hookCalled = true }

		commonChecks(3*time.Second, testErr)
		test.True(t, hookCalled)
	})
}

func TestMockComponent_WaitReady(t *testing.T) {
	synctest.Run(func() {
		mc := &MockComponent{}
		test.False(t, mc.Recorder.WaitReady.Called)

		commonChecks := func(wantD time.Duration, wantErrIs error) {
			mc.Recorder.WaitReady.Called = false
			mc.Recorder.WaitReady.Ctx = nil
			mc.Recorder.WaitReady.AbortLoopCh = nil

			abortLoopCh := make(chan struct{})

			t0 := time.Now()
			err := mc.WaitReady(t.Context(), abortLoopCh)
			d := time.Since(t0)

			test.Eq(t, wantD, d)
			test.ErrorIs(t, err, wantErrIs)

			test.True(t, mc.Recorder.WaitReady.Called)
			test.Eq(t, t.Context(), mc.Recorder.WaitReady.Ctx)
			test.Eq(t, abortLoopCh, mc.Recorder.WaitReady.AbortLoopCh)
		}

		// Should work normally with no args
		commonChecks(0, nil)

		// Our different controls should work
		testErr := errors.New("bleh")
		hookCalled := false
		mc.WaitReadyOptions.Err = testErr
		mc.WaitReadyOptions.Sleep = 4 * time.Second
		mc.WaitReadyOptions.Hook = func() { hookCalled = true }

		commonChecks(4*time.Second, testErr)
		test.True(t, hookCalled)
	})
}
