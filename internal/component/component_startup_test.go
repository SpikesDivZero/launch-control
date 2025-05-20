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
	"github.com/spikesdivzero/launch-control/internal/lcerrors"
	"github.com/spikesdivzero/launch-control/internal/testutil"
)

func TestComponent_Start(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// Happy path testing is focursed on the full lifecycle of all the things Start has to invoke, as
		// well as checking to see that the separation of concerns is honored (as best we reasonably can).
		//
		// It probably won't ever be perfect, since internal func calls can't easily be mocked out,
		// but that's OK -- it doesn't need to be.
		c := newTestingComponent(t)

		testErr := errors.New("error for ImplRun to return")

		c.ImplRun = func(ctx context.Context) error {
			<-ctx.Done()
			return testErr
		}

		exitNotifiedCh := make(chan error, 1)
		c.notifyOnExited = func(err error) {
			exitNotifiedCh <- err
			close(exitNotifiedCh)
		}

		ctx, cancel := context.WithCancelCause(t.Context())
		defer cancel(errors.New("test ended"))

		err := c.Start(ctx)
		test.ErrorIs(t, err, nil)

		// Things Start() sets up.
		must.NotNil(t, c.runCtxCancel)
		must.NotNil(t, c.doneCh)
		testutil.ChanReadIsBlocked(t, c.doneCh)

		// And our call state
		testutil.ChanReadIsBlocked(t, exitNotifiedCh)

		// Okay, it's started, and we assume the exit monitor has also started up.
		// Let's see that runCtxCancel works (and that it's piped into ImplRun)
		c.runCtxCancel()

		// It should respond to the closure within 100ms. (Perhaps I should reach for
		// synctest here for better reliability on high load machines)
		select {
		case <-time.After(100 * time.Millisecond): // FIXME: use synctest
			t.Error("ImplRun failed to respond to runCtxCancel, or doneCh wasn't closed by Start")
			return
		case <-c.doneCh:
			// Happy path
		}

		// monitorExit should detect the exit and report it within 100ms.
		select {
		case <-time.After(100 * time.Millisecond): // FIXME: use synctest
			t.Error("monitorExit failed to detect the exit status, or wasn't started by Start")
			return
		case err, ok := <-exitNotifiedCh:
			test.True(t, ok)
			test.ErrorIs(t, testErr, err)
		}
	})

	t.Run("prevent double call", func(t *testing.T) {
		defer testutil.WantPanic(t, "Start called twice?")
		c := newTestingComponent(t)
		c.doneCh = make(chan struct{})
		_ = c.Start(t.Context())
	})
}

func TestComponent_monitorExit(t *testing.T) {
	testErr := errors.New("test error 1")

	type control struct {
		c         *Component
		ctx       context.Context
		ctxCancel context.CancelFunc
		runErrCh  chan error
	}
	tests := []struct {
		name    string
		control func(control)
		wantErr error
		wantD   time.Duration
	}{
		{
			"select 1 ok nil",
			func(c control) {
				time.Sleep(time.Second)
				c.runErrCh <- nil
			},
			nil,
			time.Second,
		},
		{
			"select 1 ok err",
			func(c control) {
				time.Sleep(time.Second)
				c.runErrCh <- testErr
			},
			testErr,
			time.Second,
		},
		{
			"select 1 premature",
			func(c control) {
				time.Sleep(500 * time.Millisecond)
				close(c.runErrCh)
			},
			errPrematureChannelClose,
			500 * time.Millisecond,
		},
		// select 2 is triggered by context cancellation
		{
			"select 2 timeout",
			func(c control) {
				time.Sleep(750 * time.Millisecond)
				c.ctxCancel()
				// We never write to the channel, so it'll timeout (from this point) after the async grace period
			},
			lcerrors.ErrMonitorExitedWhileStillAlive,
			750*time.Millisecond + defaultAsyncGracePeriod,
		},

		{
			"select 2 ok nil",
			func(c control) {
				time.Sleep(5 * time.Second)
				c.ctxCancel()
				time.Sleep(defaultAsyncGracePeriod / 2)
				c.runErrCh <- nil
			},
			nil,
			5*time.Second + defaultAsyncGracePeriod/2,
		},
		{
			"select 2 ok err",
			func(c control) {
				time.Sleep(7 * time.Second)
				c.ctxCancel()
				time.Sleep(defaultAsyncGracePeriod / 2)
				c.runErrCh <- testErr
			},
			testErr,
			7*time.Second + defaultAsyncGracePeriod/2,
		},
		{
			"select 2 premature",
			func(c control) {
				time.Sleep(6 * time.Second)
				c.ctxCancel()
				time.Sleep(defaultAsyncGracePeriod / 2)
				close(c.runErrCh)
			},
			errPrematureChannelClose,
			6*time.Second + defaultAsyncGracePeriod/2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Run(func() {
				c := newTestingComponent(t)

				// We use a channel to test the result here so we can check timings.
				resultCh := make(chan error, 3)
				c.notifyOnExited = func(err error) {
					resultCh <- err
				}

				calledLogError := false
				c.logError = func(stage string, err error) {
					calledLogError = true
					test.Eq(t, "monitor-exit", stage)
					resultCh <- err // To be tested below
				}

				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()

				// The test decides what to write/close, and when.
				runErrCh := make(chan error, 1)

				go tt.control(control{
					c:         c,
					ctx:       ctx,
					ctxCancel: cancel,
					runErrCh:  runErrCh,
				})

				go func() {
					c.monitorExit(ctx, runErrCh)
					resultCh <- errors.New("test fail: neither logError nor notifyOnExited called?")
				}()

				t0 := time.Now()
				err := <-resultCh
				test.Eq(t, tt.wantD, time.Since(t0))
				test.ErrorIs(t, err, tt.wantErr)

				wantCalledLogError := tt.wantErr == lcerrors.ErrMonitorExitedWhileStillAlive
				test.Eq(t, wantCalledLogError, calledLogError)
			})
		})
	}
}
