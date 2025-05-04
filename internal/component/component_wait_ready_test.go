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
		err := c.waitReady(t.Context())
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
		err := c.waitReady(t.Context())
		test.ErrorIs(t, err, nil)
		test.Eq(t, []byte("cbc"), calls)
	})
}

func Test_waitReady_MainLoop(t *testing.T) {
	testErr1 := errors.New("fancy feast")

	type backoffReturn error
	type checkReturn struct {
		ready bool
		error error
	}

	tests := []struct {
		name           string
		maxAttempts    int
		checkReturns   []checkReturn
		backoffReturns []backoffReturn
		wantError      error
	}{
		{
			"ok, attempt 1",
			-1,
			[]checkReturn{{true, nil}},
			[]backoffReturn{},
			nil,
		},
		{
			"ok, attempt 3",
			-1,
			[]checkReturn{{false, nil}, {false, nil}, {true, nil}},
			[]backoffReturn{nil, nil},
			nil,
		},
		{
			"check says abort",
			-1,
			[]checkReturn{{false, testErr1}},
			[]backoffReturn{},
			testErr1,
		},
		{
			"backoff says abort",
			-1,
			[]checkReturn{{false, nil}, {true, nil}},
			[]backoffReturn{testErr1},
			testErr1,
		},
		{
			"bust max attempts",
			2,
			[]checkReturn{{false, nil}, {false, nil}, {true, nil}},
			[]backoffReturn{nil, nil},
			errWaitReadyExceededMaxAttempts,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Dumb mock implementations for the test
			backoffIdx, checkReadyIdx, nextCall := 0, 0, 'c'
			mockBackoff := func(ctx context.Context) error {
				must.Eq(t, 'b', nextCall)
				nextCall = 'c'

				must.Less(t, len(tt.backoffReturns), backoffIdx)
				data := tt.backoffReturns[backoffIdx]
				backoffIdx++

				test.Eq(t, t.Context(), ctx)
				return data
			}
			mockCheckReady := func(ctx context.Context) (bool, error) {
				must.Eq(t, 'c', nextCall)
				nextCall = 'b'

				must.Less(t, len(tt.checkReturns), checkReadyIdx)
				data := tt.checkReturns[checkReadyIdx]
				checkReadyIdx++

				test.Eq(t, t.Context(), ctx)
				return data.ready, data.error
			}

			if tt.maxAttempts == -1 {
				tt.maxAttempts = len(tt.checkReturns) + 1
			}
			err := waitReady_MainLoop(t.Context(), tt.maxAttempts, mockCheckReady, mockBackoff)
			test.ErrorIs(t, err, tt.wantError)
		})
	}

	/*
	   for attempt := range maxAttempts {
	           if attempt > 0 {
	                   if err := backoff(ctx); err != nil {
	                           return err
	                   }
	           }

	           if ready, err := checkOnce(ctx); ready {
	                   return nil
	           } else if err != nil {
	                   return err
	           }
	   }
	   return errWaitReadyExceededMaxAttempts
	*/
}

func TestComponent_waitReady_Backoff(t *testing.T) {
	testErr := errors.New("test err")

	type testControl struct {
		cancelCtx context.CancelCauseFunc
		closeDone func()
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

				if tt.control != nil {
					go tt.control(tc)
					synctest.Wait()
				}

				t0 := time.Now()
				err := c.waitReady_Backoff(ctx)
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
