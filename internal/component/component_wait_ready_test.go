//go:build goexperiment.synctest

package component

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/shoenig/test"
	"github.com/shoenig/test/must"
	"github.com/spikesdivzero/launch-control/internal/testutil"
)

func TestComponent_waitReady(t *testing.T) {
	// This is just a minimal test to ensure that it passes args right. More detailed tests belong elsewhere.
	t.Run("no impl", func(t *testing.T) {
		c := newTestingComponent(t)
		c.ImplCheckReady = nil
		err := c.WaitReady(t.Context(), make(chan struct{}))
		test.Nil(t, err)
	})

	t.Run("basic calls", func(t *testing.T) {
		c := newTestingComponent(t)
		calls := []byte{}
		c.ImplCheckReady = func(ctx context.Context) (bool, error) {
			calls = append(calls, 'c')
			switch len(calls) {
			case 1:
				return false, nil
			case 3:
				return true, nil
			default:
				t.Errorf("ImplCheckReady shouldn't be called at idx %v", len(calls))
				t.FailNow()
			}
			panic("unreachable")
		}
		c.CheckReadyOptions.Backoff = func() time.Duration {
			calls = append(calls, 'b')
			return 0
		}
		err := c.WaitReady(t.Context(), make(chan struct{}))
		test.ErrorIs(t, err, nil)
		test.Eq(t, []byte("cbc"), calls)
	})
}

func Test_waitReady_MainLoop(t *testing.T) {
	testErr1 := errors.New("fancy feast")

	// The basic check event sequence for the loop is: Ready, Abort, Backoff, Abort
	//
	// At the start of each checkReady call, we ask control for what to return this time around.
	type loopControl struct {
		Ready      bool
		ReadyError error

		AbortBeforeBackoff bool

		BackoffError error

		AbortAfterBackoff bool

		isValid bool // Used by the test mock impls
	}

	tests := []struct {
		name        string
		maxAttempts int
		controls    []loopControl
		wantError   error
	}{
		{
			"ok, attempt 1",
			-1,
			[]loopControl{
				{Ready: true},
			},
			nil,
		},
		{
			"ok, attempt 3",
			-1,
			[]loopControl{
				{Ready: false},
				{Ready: false},
				{Ready: true},
			},
			nil,
		},
		{
			"check says abort",
			-1,
			[]loopControl{
				{ReadyError: testErr1},
			},
			testErr1,
		},
		{
			"backoff says abort",
			-1,
			[]loopControl{
				{BackoffError: testErr1},
			},
			testErr1,
		},
		{
			"bust max attempts",
			2,
			[]loopControl{
				{Ready: false},
				{Ready: false},
				{Ready: true},
			},
			errWaitReadyExceededMaxAttempts,
		},
		{
			"abort chan closed before backoff",
			-1,
			[]loopControl{
				{Ready: false},
				{AbortBeforeBackoff: true},
				{Ready: true},
			},
			errWaitReadyAbortChClosed,
		},
		{
			"abort chan closed after backoff",
			-1,
			[]loopControl{
				{Ready: false},
				{AbortAfterBackoff: true},
				{Ready: true},
			},
			errWaitReadyAbortChClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Dumb mock implementations for the test
			var control loopControl
			controlIdx, nextCall := 0, 'c'
			abortLoopCh, closeAbortLoopCh := testutil.ChanWithCloser[struct{}](0)

			mockBackoff := func(ctx context.Context, innerAbortCh <-chan struct{}) error {
				must.Eq(t, 'b', nextCall)
				nextCall = 'c'

				must.True(t, control.isValid) // overrun of control inputs
				if control.AbortAfterBackoff {
					closeAbortLoopCh()
				}

				test.Eq(t, t.Context(), ctx)
				test.Eq(t, abortLoopCh, innerAbortCh)
				return control.BackoffError
			}
			mockCheckReady := func(ctx context.Context) (bool, error) {
				must.Eq(t, 'c', nextCall)
				nextCall = 'b'

				must.Less(t, len(tt.controls), controlIdx)
				control = tt.controls[controlIdx]
				controlIdx++
				control.isValid = true

				if control.AbortBeforeBackoff {
					closeAbortLoopCh()
				}

				test.Eq(t, t.Context(), ctx)
				return control.Ready, control.ReadyError
			}

			if tt.maxAttempts == -1 {
				tt.maxAttempts = len(tt.controls) + 1
			}
			err := waitReady_MainLoop(t.Context(), abortLoopCh, tt.maxAttempts, mockCheckReady, mockBackoff)
			test.ErrorIs(t, err, tt.wantError)
		})
	}
}

func TestComponent_waitReady_Backoff(t *testing.T) {
	testErr := errors.New("test err")

	type testControl struct {
		cancelCtx  context.CancelCauseFunc
		closeDone  func()
		closeAbort func()
	}

	type want struct {
		err error
		d   time.Duration
	}

	tests := []struct {
		name    string
		backoff time.Duration
		control func(testControl)
		want    want
	}{
		{
			"zero delay",
			0,
			nil,
			want{nil, 0},
		},
		{
			"negative delay",
			-12,
			nil,
			want{nil, 0},
		},
		{
			"normal delay",
			3 * time.Second,
			nil,
			want{nil, 3 * time.Second},
		},
		{
			"interrupt: ctx",
			3 * time.Second,
			func(tc testControl) {
				time.Sleep(time.Second)
				tc.cancelCtx(testErr)
			},
			want{testErr, time.Second},
		},
		{
			"intterupt: exited",
			3 * time.Second,
			func(tc testControl) {
				time.Sleep(2 * time.Second)
				tc.closeDone()
			},
			want{errWaitReadyComponentExited, 2 * time.Second},
		},
		{
			"interrupt: abort",
			5 * time.Second,
			func(tc testControl) {
				time.Sleep(3 * time.Second)
				tc.closeAbort()
			},
			want{errWaitReadyAbortChClosed, 3 * time.Second},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Run(func() {
				c := newTestingComponent(t)
				tc := testControl{}

				c.CheckReadyOptions.Backoff = func() time.Duration { return tt.backoff }

				var ctx context.Context
				ctx, tc.cancelCtx = context.WithCancelCause(t.Context())
				defer tc.cancelCtx(errors.New("test ended"))

				c.doneCh, tc.closeDone = testutil.ChanWithCloser[struct{}](0)

				var abortCh chan struct{}
				abortCh, tc.closeAbort = testutil.ChanWithCloser[struct{}](0)

				if tt.control != nil {
					go tt.control(tc)
					synctest.Wait()
				}

				t0 := time.Now()
				err := c.waitReady_Backoff(ctx, abortCh)
				test.ErrorIs(t, err, tt.want.err)
				test.Eq(t, tt.want.d, time.Since(t0))
			})
		})
	}
}

func TestComponent_waitReady_CheckOnce(t *testing.T) {
	type testControl struct {
		c         *Component
		closeDone func()
		cancelCtx context.CancelCauseFunc
	}

	type checkReturn struct {
		ready bool
		error error
		d     time.Duration
	}

	type wantResult struct {
		ready bool
		error error
		d     time.Duration
	}

	tests := []struct {
		name        string
		control     func(testControl)
		checkReturn checkReturn
		want        wantResult
	}{
		{
			"already exited",
			func(tc testControl) {
				tc.closeDone()
				tc.c.ImplCheckReady = func(ctx context.Context) (bool, error) { panic("should not be called") }
			},
			checkReturn{}, // unused
			wantResult{false, errWaitReadyComponentExited, 0},
		},
		{
			"good call, result=true, no err",
			nil,
			checkReturn{true, nil, 0},
			wantResult{true, nil, 0},
		},
		{
			"good call, result=false, no err",
			nil,
			checkReturn{false, nil, 0},
			wantResult{false, nil, 0},
		},
		{
			// We log user errors, but don't return them to the main loop.
			"good call, result=false, generic user-error",
			nil,
			checkReturn{false, errors.New("test user error"), 0},
			wantResult{false, nil, 0},
		},
		{
			"call timeout",
			func(tc testControl) { tc.c.CheckReadyOptions.CallTimeout = time.Second },
			checkReturn{true, nil, 2 * time.Second},
			// FIXME: hard-coded 100ms grace period
			wantResult{false, context.DeadlineExceeded, time.Second + 100*time.Millisecond},
		},
		{
			"interrupt: run exits",
			func(tc testControl) {
				time.Sleep(time.Second)
				tc.closeDone()
			},
			checkReturn{true, nil, 2 * time.Second},
			wantResult{false, errWaitReadyComponentExited, time.Second},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Run(func() {
				c := newTestingComponent(t)
				tc := testControl{c: c}

				c.ImplCheckReady = func(ctx context.Context) (bool, error) {
					ret := tt.checkReturn
					if ret.d > 0 {
						time.Sleep(ret.d)
					}
					return ret.ready, ret.error
				}

				// TODO: should we check this?
				c.logError = func(string, error) {}

				var ctx context.Context
				ctx, tc.cancelCtx = context.WithCancelCause(t.Context())
				defer tc.cancelCtx(errors.New("test done"))

				c.doneCh, tc.closeDone = testutil.ChanWithCloser[struct{}](0)

				if tt.control != nil {
					go tt.control(tc)
					synctest.Wait()
				}

				t0 := time.Now()
				ready, err := c.waitReady_CheckOnce(ctx)
				test.Eq(t, tt.want.ready, ready)
				test.ErrorIs(t, err, tt.want.error)
				test.Eq(t, tt.want.d, time.Since(t0))
			})
		})
	}
}
