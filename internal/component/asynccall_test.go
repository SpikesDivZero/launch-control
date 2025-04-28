package component

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"testing/synctest"
	"time"

	"github.com/shoenig/test"
)

type dummyError string

func (s dummyError) Error() string { return string(s) }

func TestAsyncCall(t *testing.T) {
	type control struct {
		ctxCancel context.CancelCauseFunc
	}
	type args struct {
		timeout      time.Duration
		timeoutGrace time.Duration
		f            func(context.Context) int
	}
	type want struct {
		val int
		err error
		d   time.Duration
	}
	tests := []struct {
		name    string
		args    args
		control func(control)
		want    want
	}{
		{ // When the parent context is already dead, we shouldn't invoke the user func
			"parent ctx already dead",
			args{
				time.Second, time.Second,
				func(ctx context.Context) int { panic("shouldn't be called") },
			},
			func(c control) {
				c.ctxCancel(dummyError("parent dead"))
			},
			want{0, dummyError("parent dead"), 0},
		},
		{ // User function returns immediately
			"user fast return",
			args{
				time.Second, time.Second,
				func(ctx context.Context) int { return 84 },
			},
			nil,
			want{84, nil, 0},
		},
		{ // User function returns after a short timeout
			// Similar to above, but checks our in-test timing closer before we start digging any deeper.
			"user slow but good return",
			args{
				5 * time.Second, time.Second,
				func(ctx context.Context) int {
					time.Sleep(2 * time.Second)
					return 63
				},
			},
			nil,
			want{63, nil, 2 * time.Second},
		},
		{ // User function exits without writing a value (only possible via runtime.Goexit?)
			"user causes goexit",
			args{
				3 * time.Second, time.Second,
				func(ctx context.Context) int {
					time.Sleep(time.Second)
					runtime.Goexit()
					panic("unreachable")
				},
			},
			nil,
			want{0, errPrematureChannelClose, time.Second},
		},
		{
			"timeout, no grace",
			args{
				time.Second, 0,
				func(ctx context.Context) int {
					<-ctx.Done()
					return 67
				},
			},
			nil,
			want{0, context.DeadlineExceeded, time.Second},
		},
		{
			"timeout, no luck in grace",
			args{
				4 * time.Second, time.Second,
				func(ctx context.Context) int {
					time.Sleep(6 * time.Second)
					return 64
				},
			},
			nil,
			want{0, context.DeadlineExceeded, 5 * time.Second},
		},
		{
			"timeout, result in grace",
			args{
				8 * time.Second, 2 * time.Second,
				func(ctx context.Context) int {
					<-ctx.Done()
					time.Sleep(time.Second)
					return 96
				},
			},
			nil,
			want{96, nil, 9 * time.Second},
		},
		{
			"timeout, user goexit in grace",
			args{
				3 * time.Second, 2 * time.Second,
				func(ctx context.Context) int {
					time.Sleep(4 * time.Second)
					runtime.Goexit()
					panic("unreachable")
				},
			},
			nil,
			want{0, errPrematureChannelClose, 4 * time.Second},
		},
		{
			"parent cancel cause honored",
			args{
				time.Minute, 0,
				func(ctx context.Context) int {
					<-ctx.Done()
					time.Sleep(time.Second)
					return 12
				},
			},
			func(c control) {
				time.Sleep(time.Second)
				c.ctxCancel(dummyError("test parent cancel"))
			},
			want{0, dummyError("test parent cancel"), time.Second},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Run(func() {
				ctx, cancelCause := context.WithCancelCause(t.Context())
				defer cancelCause(errors.New("test ended"))

				if tt.control != nil {
					go tt.control(control{
						ctxCancel: cancelCause,
					})
					synctest.Wait()
				}

				t0 := time.Now()
				ch := AsyncCall(ctx, tt.args.timeout, tt.args.f, tt.args.timeoutGrace)
				got, err := (<-ch).Values()
				gotD := time.Since(t0)

				test.Eq(t, tt.want.d, gotD)
				test.Eq(t, tt.want.val, got)
				test.ErrorIs(t, tt.want.err, err)
			})
		})
	}
}
