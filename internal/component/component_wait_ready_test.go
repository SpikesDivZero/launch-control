package component

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/shoenig/test"
)

// Runs the provided function inside of synctest, returning the time it took for it to execute.
func syncTimeIt(t *testing.T, f func(*testing.T)) (d time.Duration) {
	synctest.Test(t, func(t *testing.T) {
		t0 := time.Now()
		f(t)
		d = time.Since(t0)
	})
	return d
}

// Here, our main focus is on testing that we wire things into the correct slot of waitReady_loop.
//
// More extensive tests will show up there.
func TestComponent_waitReady(t *testing.T) {
	type mockCheckReturn struct {
		ok  bool
		err error
	}

	testErr := errors.New("in test")

	for _, tt := range []struct {
		name              string
		mockCheckReturn   []mockCheckReturn // if nil, Impl func is not created
		mockBackoffReturn []time.Duration   // if nil, Impl func is not created
		expectD           time.Duration
		expectStopErr     error
	}{
		{ // Do not crash when no check function provided
			"no check func",
			[]mockCheckReturn(nil),
			[]time.Duration(nil),
			0,
			nil,
		},
		{ // If no backoff function is provided, default to one that returns no delay
			"no backoff = default 0",
			[]mockCheckReturn{{false, nil}, {false, nil}, {true, nil}},
			[]time.Duration(nil),
			0,
			nil,
		},
		{ // If one is provided, then use that.
			"has backoff",
			[]mockCheckReturn{{false, nil}, {false, nil}, {true, nil}},
			[]time.Duration{time.Second, time.Minute},
			61 * time.Second,
			nil,
		},
		{ // Ensure we're passing through RequestStop in the right slot.
			"error triggers stop request",
			[]mockCheckReturn{{false, testErr}},
			nil,
			0,
			testErr,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gotD := syncTimeIt(t, func(t *testing.T) {
				c := &Component{
					// Not populating base ctx as I don't want it used in wait-ready
					runCtx: t.Context(),
				}

				checkCalls := 0
				if tt.mockCheckReturn != nil {
					c.ImplCheckReady = func(ctx context.Context) (bool, error) {
						md := tt.mockCheckReturn[checkCalls]
						checkCalls++
						return md.ok, md.err
					}
				}

				backoffCalls := 0
				if tt.mockBackoffReturn != nil {
					c.ImplCheckReadyBackoff = func() time.Duration {
						md := tt.mockBackoffReturn[backoffCalls]
						backoffCalls++
						return md
					}
				}

				var gotStopErr error
				c.callbacks.RequestStop = func(gotC *Component, reason error) {
					test.Eq(t, c, gotC)
					gotStopErr = reason
				}

				// TODO: Test that we're plumbing the start/waitReady context, and not the run context.
				c.waitReady(t.Context())

				test.Eq(t, len(tt.mockCheckReturn), checkCalls)
				test.Eq(t, len(tt.mockBackoffReturn), backoffCalls)
				test.Eq(t, tt.expectStopErr, gotStopErr)
			})

			test.Eq(t, tt.expectD, gotD)
		})
	}
}

func TestComponent_waitReady_loop(t *testing.T) {
	type mockCheckReturn struct {
		ok    bool
		err   error
		abort bool
	}
	type mockBackoffReturn struct {
		d     time.Duration
		abort bool
	}

	testErr := errors.New("test")
	_ = testErr

	for _, tt := range []struct {
		name              string
		mockCheckReturn   []mockCheckReturn
		mockBackoffReturn []mockBackoffReturn
		control           func(doAbort func())
		expectD           time.Duration
		expectStopErr     error
		expectReady       bool
	}{
		{ // If we enter the loop and we're already aborting, then we shouldn't call anything
			"skip if already aborting at start",
			[]mockCheckReturn{},
			[]mockBackoffReturn{},
			func(doAbort func()) { doAbort() },
			0,
			nil,
			false,
		},
		{
			"normal loop until ready",
			[]mockCheckReturn{{}, {}, {ok: true}},
			[]mockBackoffReturn{{d: time.Second}, {d: 3 * time.Second}},
			nil,
			4 * time.Second,
			nil,
			true,
		},
		{ // If an abort happens during our check, then we shouldn't proceed to backoff again
			"abort during check ready",
			[]mockCheckReturn{{}, {abort: true}},
			[]mockBackoffReturn{{d: 3 * time.Second}}, // a second call will cause a t.Error
			nil,
			3 * time.Second,
			nil,
			false,
		},
		{ // If an abort happens during our backoff, then we shouldn't proceed to check again.
			"abort during backoff",
			[]mockCheckReturn{{}}, // a second call will cause a t.Error
			[]mockBackoffReturn{{abort: true}},
			nil,
			0,
			nil,
			false,
		},
		{ // If our check function returns an error, we should report it and stop checking
			"check returns error",
			[]mockCheckReturn{{}, {err: testErr}},
			[]mockBackoffReturn{{}},
			nil,
			0,
			testErr,
			false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gotD := syncTimeIt(t, func(t *testing.T) {
				runCtx, runCtxCancel := context.WithCancel(t.Context())
				defer runCtxCancel()

				startCtx, startCtxCancel := context.WithCancel(t.Context())
				defer startCtxCancel()

				c := &Component{runCtx: runCtx}

				checkCalls := 0
				fnCheck := func(ctx context.Context) (bool, error) {
					if checkCalls == len(tt.mockCheckReturn) {
						err := errors.New("mock ImplCheckReady called more times than plan allows")
						t.Error(err)
						return false, err
					}

					// The check function should get the run context (not the start context)
					test.EqOp(t, runCtx, ctx)

					md := tt.mockCheckReturn[checkCalls]
					checkCalls++

					if md.abort {
						startCtxCancel()
						synctest.Wait()
					}
					return md.ok, md.err
				}

				backoffCalls := 0
				fnCheckBackoff := func() time.Duration {
					if backoffCalls == len(tt.mockBackoffReturn) {
						err := errors.New("mock fnBackoff called more times than plan allows")
						t.Error(err)
						return -1
					}

					md := tt.mockBackoffReturn[backoffCalls]
					backoffCalls++

					if md.abort {
						startCtxCancel()
						synctest.Wait()
					}
					return md.d
				}

				var gotStopErr error
				fnRequestStop := func(gotC *Component, reason error) {
					test.Eq(t, c, gotC)
					gotStopErr = reason
				}

				if tt.control != nil {
					tt.control(startCtxCancel)
					synctest.Wait()
				}

				c.waitReady_loop(startCtx,
					fnCheck,
					fnCheckBackoff,
					fnRequestStop)

				test.Eq(t, len(tt.mockCheckReturn), checkCalls)
				test.Eq(t, len(tt.mockBackoffReturn), backoffCalls)
				test.Eq(t, tt.expectStopErr, gotStopErr)
			})

			test.Eq(t, tt.expectD, gotD)
		})
	}
}

func TestComponent_waitReady_createShouldAbort(t *testing.T) {
	isChanClosed := func(ch <-chan struct{}) bool {
		select {
		case <-ch:
			return true
		default:
			return false
		}
	}

	type testControl struct {
		cancelStartCtx context.CancelFunc
		cancelRunCtx   context.CancelFunc
		closeRunExited func()
	}

	for _, tt := range []struct {
		name       string
		immediate  bool // Should we run causeAbort before calling createShouldAbort
		causeAbort func(tc testControl)
	}{
		{
			"immediate: start ctx cancel",
			true,
			func(tc testControl) { tc.cancelStartCtx() },
		},
		{
			"immediate: run ctx cancel",
			true,
			func(tc testControl) { tc.cancelRunCtx() },
		},
		{
			"immediate: responds to run exited",
			true,
			func(tc testControl) { tc.closeRunExited() },
		},
		{
			"normal: start ctx cancel",
			false,
			func(tc testControl) { tc.cancelStartCtx() },
		},
		{
			"normal: run ctx cancel",
			false,
			func(tc testControl) { tc.cancelRunCtx() },
		},
		{
			"normal: responds to run exited",
			false,
			func(tc testControl) { tc.closeRunExited() },
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				runCtx, runCtxCancel := context.WithCancel(t.Context())
				defer runCtxCancel()

				startCtx, startCtxCancel := context.WithCancel(t.Context())
				defer startCtxCancel()

				runExitedCh := make(chan struct{})

				tc := testControl{
					cancelStartCtx: startCtxCancel,
					cancelRunCtx:   runCtxCancel,
					closeRunExited: func() { close(runExitedCh) },
				}

				if tt.immediate {
					tt.causeAbort(tc)
					synctest.Wait()
				}

				c := &Component{
					runCtx:      runCtx,
					runExitedCh: runExitedCh,
				}
				abortCh, shouldAbort := c.waitReady_createShouldAbort(startCtx)

				test.EqOp(t, tt.immediate, isChanClosed(abortCh))
				test.EqOp(t, tt.immediate, shouldAbort())

				if !tt.immediate {
					tt.causeAbort(tc)
					synctest.Wait()
				}

				test.True(t, isChanClosed(abortCh))
				test.True(t, shouldAbort())
			})

		})
	}
}

func TestComponent_waitReady_sleep(t *testing.T) {
	for _, tt := range []struct {
		name    string
		sleepD  time.Duration
		expectD time.Duration
		control func(abortCh chan struct{})
	}{
		{"neg", -1, 0, nil},
		{"zero", 0, 0, nil},
		{"pos", time.Minute, time.Minute, nil},
		{
			"interrupt: immediate",
			time.Hour,
			0,
			func(abortCh chan struct{}) {
				close(abortCh)
			},
		},
		{
			"interrupt: delayed",
			time.Hour,
			3 * time.Minute,
			func(abortCh chan struct{}) {
				time.Sleep(3 * time.Minute)
				close(abortCh)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				c := &Component{}
				abortCh := make(chan struct{})

				if tt.control != nil {
					go tt.control(abortCh)
					synctest.Wait()
				}

				t0 := time.Now()
				c.waitReady_sleep(tt.sleepD, abortCh)

				test.Eq(t, tt.expectD, time.Since(t0))
			})
		})
	}
}
