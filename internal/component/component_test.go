package component

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/shoenig/test"
	"github.com/shoenig/test/must"
	"github.com/spikesdivzero/launch-control/internal/testutil"
)

func TestComponent_Register(t *testing.T) {
	// We trust that most of it is valid, since the types are different so errors are unlikely.
	// We'll check the basics, as well as the context chain making sense.

	ctxKey := &struct{ _ int }{42}
	ctx := context.WithValue(t.Context(), ctxKey, any(64))
	log := slog.New(slog.DiscardHandler)

	c := &Component{Name: "test"}

	notifyCalled := false
	c.Register(ctx, log, ControllerCallbacks{
		RequestStop: func(err error) { notifyCalled = true },
	})

	test.EqOp(t, log, c.log)

	must.NotNil(t, c.callbacks.RequestStop)
	c.callbacks.RequestStop(nil)
	test.True(t, notifyCalled)

	// check the ctx chain
	must.NotNil(t, c.ctx)
	must.NotNil(t, c.runCtx)

	// They should all be different
	test.NotEq(t, ctx, c.ctx)
	test.NotEq(t, ctx, c.runCtx)
	test.NotEq(t, c.ctx, c.runCtx)

	// And they should all be in the chain
	test.Eq(t, any(64), ctx.Value(ctxKey))
	test.Eq(t, any(64), c.ctx.Value(ctxKey))
	test.Eq(t, any(64), c.runCtx.Value(ctxKey))

	// The runCtx must be a child of the main ctx. We'll check this via cancel propegation
	cancelErr := errors.New("e1")
	c.ctxCancel(cancelErr)

	test.Nil(t, ctx.Err())
	test.EqOp(t, cancelErr, context.Cause(c.ctx))
	test.EqOp(t, cancelErr, context.Cause(c.runCtx))
}

func TestComponent_Start(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := &Component{
			Name: "test",
		}

		startCtx, startCtxCancel := context.WithCancel(t.Context())
		defer startCtxCancel()

		startedCh := make(chan struct{})
		c.ImplRun = func(ctx context.Context) error {
			close(startedCh)
			<-c.runCtx.Done()

			test.EqOp(t, c.runCtx, ctx)

			return nil
		}

		checkedReady := false
		c.ImplCheckReady = func(ctx context.Context) (bool, error) {
			checkedReady = true

			test.EqOp(t, c.runCtx, ctx)

			return true, nil
		}

		notifiedCh := make(chan struct{})
		c.Register(t.Context(), slog.New(slog.DiscardHandler), ControllerCallbacks{
			RequestStop: func(err error) {
				close(notifiedCh)
			},
		})

		// TODO: Test that we're plumbing the startCtx all the way down (separate from the runCtx)
		c.Start(startCtx)
		synctest.Wait()

		testutil.ChanReadIsClosed(t, startedCh, test.Sprint("ImplRun started"))
		test.True(t, checkedReady)

		c.runCtxCancel(nil)
		synctest.Wait()

		testutil.ChanReadIsClosed(t, notifiedCh, test.Sprint("monitorExit returned"))
	})
}

func TestComponent_monitorExit(t *testing.T) {
	c := &Component{Name: "test"}

	var wantErr error
	var calledNotify, wantCalledNotify int
	c.callbacks.RequestStop = func(gotErr error) {
		calledNotify++
		test.ErrorIs(t, gotErr, wantErr)
	}
	defer func() {
		test.Eq(t, calledNotify, wantCalledNotify)
	}()

	t.Run("no error", func(t *testing.T) {
		wantCalledNotify++
		wantErr = nil

		ch := make(chan error, 1)
		ch <- nil
		close(ch)

		c.monitorExit(ch)
	})

	t.Run("error", func(t *testing.T) {
		wantCalledNotify++
		wantErr = errors.New("e1")

		ch := make(chan error, 1)
		ch <- wantErr
		close(ch)

		c.monitorExit(ch)
	})

	t.Run("premature close", func(t *testing.T) {
		wantCalledNotify++
		wantErr = PrematureChannelCloseError{"resultCh"}

		ch := make(chan error, 1)
		// We don't write any message, as if the user code called runtime.Goexit
		close(ch)

		c.monitorExit(ch)
	})
}

func TestComponent_Stop(t *testing.T) {
	err1 := errors.New("test err")

	tests := []struct {
		name      string
		runWaitD  time.Duration
		stopWaitD time.Duration
		stopErr   error
		wantD     time.Duration
	}{
		{
			"happy, run returns first",
			3 * time.Second,
			5 * time.Second,
			nil,
			5 * time.Second,
		},
		{
			"happy, stop returns first",
			8 * time.Second,
			2 * time.Second,
			nil,
			8 * time.Second,
		},
		{
			"happy-ish, stop returns error",
			time.Second,
			time.Second,
			err1,
			time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotD := syncTimeIt(t, func(t *testing.T) {
				c := &Component{
					Name: "test",
				}

				c.Register(t.Context(), nil, ControllerCallbacks{
					RequestStop: func(reason error) {}, // We're not testing monitorExit here
					ComponentError: func(gotErr error) {
						test.ErrorIs(t, gotErr, WrapComponentError(c, "stop", tt.stopErr))
					},
				})

				requestStopCh := make(chan struct{})
				c.ImplRun = func(ctx context.Context) error {
					<-requestStopCh
					time.Sleep(tt.runWaitD)
					return nil
				}

				c.ImplStop = func(ctx context.Context) error {
					close(requestStopCh)
					time.Sleep(tt.stopWaitD)
					return tt.stopErr
				}

				c.Start(t.Context())
				c.Stop()
			})

			test.Eq(t, tt.wantD, gotD)
		})
	}
}
