package component

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"testing/synctest"
	"time"

	"github.com/shoenig/test"
	"github.com/spikesdivzero/launch-control/internal/lcerrors"
)

func TestAsyncCall(t *testing.T) {
	testErrParentDead := errors.New("parent dead")

	type control struct {
		ctxCancel context.CancelCauseFunc
	}
	type timeoutArgs struct {
		d      time.Duration
		grace  time.Duration
		source string
	}
	type want struct {
		val int
		err error
		d   time.Duration
	}
	tests := []struct {
		name    string
		timeout timeoutArgs
		f       func(context.Context) int
		control func(control)
		want    want
	}{
		{ // When the parent context is already dead, we shouldn't invoke the user func
			"parent ctx already dead",
			timeoutArgs{d: time.Second, grace: time.Second},
			func(ctx context.Context) int { panic("shouldn't be called") },
			func(c control) {
				c.ctxCancel(testErrParentDead)
			},
			want{0, testErrParentDead, 0},
		},
		{ // User function returns immediately
			"user fast return",
			timeoutArgs{d: time.Second, grace: time.Second},
			func(ctx context.Context) int { return 84 },
			nil,
			want{84, nil, 0},
		},
		{ // User function returns after a short timeout
			// Similar to above, but checks our in-test timing closer before we start digging any deeper.
			"user slow but good return",
			timeoutArgs{d: 5 * time.Second, grace: time.Second},
			func(ctx context.Context) int {
				time.Sleep(2 * time.Second)
				return 63
			},
			nil,
			want{63, nil, 2 * time.Second},
		},
		{ // User function exits without writing a value (only possible via runtime.Goexit?)
			"user causes goexit",
			timeoutArgs{d: 3 * time.Second, grace: time.Second},
			func(ctx context.Context) int {
				time.Sleep(time.Second)
				runtime.Goexit()
				panic("unreachable")
			},
			nil,
			want{0, errPrematureChannelClose, time.Second},
		},
		{
			"timeout, no grace",
			timeoutArgs{d: time.Second},
			func(ctx context.Context) int {
				<-ctx.Done()
				return 67
			},
			nil,
			want{0, context.DeadlineExceeded, time.Second},
		},
		{
			"timeout, no luck in grace",
			timeoutArgs{d: 4 * time.Second, grace: time.Second},
			func(ctx context.Context) int {
				time.Sleep(6 * time.Second)
				return 64
			},
			nil,
			want{0, context.DeadlineExceeded, 5 * time.Second},
		},
		{
			"timeout, result in grace",
			timeoutArgs{d: 8 * time.Second, grace: 2 * time.Second},
			func(ctx context.Context) int {
				<-ctx.Done()
				time.Sleep(time.Second)
				return 96
			},
			nil,
			want{96, nil, 9 * time.Second},
		},
		{
			"timeout, user goexit in grace",
			timeoutArgs{d: 3 * time.Second, grace: 2 * time.Second},
			func(ctx context.Context) int {
				time.Sleep(4 * time.Second)
				runtime.Goexit()
				panic("unreachable")
			},
			nil,
			want{0, errPrematureChannelClose, 4 * time.Second},
		},
		{
			"parent cancel cause honored",
			timeoutArgs{d: time.Minute},
			func(ctx context.Context) int {
				<-ctx.Done()
				time.Sleep(time.Second)
				return 12
			},
			func(c control) {
				time.Sleep(time.Second)
				c.ctxCancel(testErrParentDead)
			},
			want{0, testErrParentDead, time.Second},
		},
		{
			"uses custom timeout error type",
			timeoutArgs{d: time.Second, source: "custom-test-source"},
			func(ctx context.Context) int {
				<-ctx.Done()
				time.Sleep(time.Second)
				return 12
			},
			nil,
			want{0, lcerrors.ContextTimeoutError{Source: "custom-test-source"}, time.Second},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancelCause := context.WithCancelCause(t.Context())
				defer cancelCause(errors.New("test ended"))

				if tt.control != nil {
					go tt.control(control{
						ctxCancel: cancelCause,
					})
					synctest.Wait()
				}

				if tt.timeout.source == "" {
					tt.timeout.source = "in-test"
				}

				t0 := time.Now()
				ch := AsyncCall(ctx, tt.timeout.source, tt.timeout.d, tt.timeout.grace, tt.f)
				got, err := (<-ch).Values()
				gotD := time.Since(t0)

				test.Eq(t, tt.want.d, gotD)
				test.Eq(t, tt.want.val, got)
				test.ErrorIs(t, err, tt.want.err)

				// HACK(go1.25 upgrade): some of our test coroutines run longer than our main test, causing a panic.
				// Sleep at end fixes this, for now. I should redo this later on to be smarter.
				time.Sleep(5 * time.Minute)
			})
		})
	}
}
