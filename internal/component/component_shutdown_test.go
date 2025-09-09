package component

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/shoenig/test"
	"github.com/spikesdivzero/launch-control/internal/lcerrors"
	"github.com/spikesdivzero/launch-control/internal/testutil"
)

func TestComponent_Shutdown(t *testing.T) {
	for _, wantErr := range []bool{false, true} {
		t.Run(fmt.Sprintf("wantErr %v", wantErr), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(t.Context())
				defer cancel(errors.New("test done"))

				c := newTestingComponent(t)

				var closeDone func()
				c.doneCh, closeDone = testutil.ChanWithCloser[struct{}](0)

				// Test goal is only to ensure we call both shutdownVia methods, and that we return an error
				// if the shutdown did not result in doneCh being closed.
				//
				// Specific implementation details of the shutdown methods are out of scope for the main func
				calls := []string{}
				c.ImplShutdown = func(ctx context.Context) error {
					calls = append(calls, "ImplShutdown")
					return nil
				}
				c.runCtxCancel = func() {
					calls = append(calls, "runCtxCancel")
					if !wantErr {
						closeDone()
					}
				}

				c.logError = func(string, error) {} // We validate our calls to this elsewhere

				err := c.Shutdown(ctx)
				test.Eq(t, []string{"ImplShutdown", "runCtxCancel"}, calls)
				if wantErr {
					test.Error(t, err)
				} else {
					test.NoError(t, err)
				}
			})
		})
	}
}

func TestComponent_isDead(t *testing.T) {
	c := newTestingComponent(t)

	var closeDone func()
	c.doneCh, closeDone = testutil.ChanWithCloser[struct{}](0)

	test.False(t, c.isDead())
	closeDone()
	test.True(t, c.isDead())
}

func TestComponent_shutdownViaImpl(t *testing.T) {
	type control struct {
		c         *Component
		closeDone func()
		cancelCtx context.CancelCauseFunc
	}
	type shutdownMock struct {
		d          time.Duration
		closesDone bool
		err        error
	}

	testErr := errors.New("user error")

	tests := []struct {
		name     string
		control  func(control)
		shutdown shutdownMock
		wantD    time.Duration
		wantLog  error
	}{
		{
			"already dead",
			func(c control) { c.closeDone() },
			shutdownMock{d: 5 * time.Second},
			0,
			nil,
		},
		{
			"run exits before ImplShutdown returns",
			nil,
			shutdownMock{d: 850 * time.Millisecond, closesDone: true},
			850 * time.Millisecond,
			nil,
		},
		{
			"ImplShutdown returns before run exits",
			func(c control) {
				time.Sleep(773 * time.Millisecond)
				c.closeDone()
			},
			shutdownMock{d: 500 * time.Millisecond},
			773 * time.Millisecond,
			nil,
		},
		{
			"ImplShutdown has a user-error", // will log, but not panic
			nil,
			shutdownMock{err: testErr, closesDone: true},
			0,
			testErr,
		},
		{
			"ImplShutdown times out",
			func(c control) {
				c.c.ShutdownOptions.CallTimeout = 3 * time.Second
				time.Sleep(4 * time.Second)
				c.closeDone()
			},
			shutdownMock{d: 5 * time.Second},
			4 * time.Second,
			lcerrors.ContextTimeoutError{Source: "Shutdown.CallTimeout"},
		},
		{
			"shutdown process fails, hits completion timeout",
			func(c control) {
				c.c.ShutdownOptions.CompletionTimeout = 2 * time.Second
			},
			shutdownMock{d: 3 * time.Second},
			2*time.Second + defaultAsyncGracePeriod,
			lcerrors.ContextTimeoutError{Source: "Shutdown.CompletionTimeout"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(t.Context())
				defer cancel(errors.New("test done"))

				c := newTestingComponent(t)
				ctrl := control{c: c, cancelCtx: cancel}

				c.doneCh, ctrl.closeDone = testutil.ChanWithCloser[struct{}](0)

				shutdownCalled := false
				c.ImplShutdown = func(ctx context.Context) error {
					shutdownCalled = true
					time.Sleep(tt.shutdown.d)
					if tt.shutdown.closesDone {
						ctrl.closeDone()
					}
					return tt.shutdown.err
				}

				// TODO: should we check this?
				logErrorCalled := false
				c.logError = func(stage string, err error) {
					logErrorCalled = true
					test.Eq(t, "shutdown (impl)", stage)
					test.Error(t, err)
					test.ErrorIs(t, err, tt.wantLog)
				}

				if tt.control != nil {
					go tt.control(ctrl)
					synctest.Wait()
				}

				t0 := time.Now()
				c.shutdownViaImpl(ctx)
				test.Eq(t, tt.wantD, time.Since(t0))

				wantShutdownCalled := tt.name != "already dead" // So sue me...
				test.Eq(t, wantShutdownCalled, shutdownCalled)

				wantLogErrorCalled := tt.wantLog != nil
				test.Eq(t, wantLogErrorCalled, logErrorCalled)

				// HACK(go1.25 upgrade): some of our test coroutines run longer than our main test, causing a panic.
				// Sleep at end fixes this, for now. I should redo this later on to be smarter.
				time.Sleep(5 * time.Minute)
			})
		})
	}
}

func TestComponent_shutdownViaContext(t *testing.T) {
	type control struct {
		closeDone func()
	}

	tests := []struct {
		name    string
		control func(control)
		wantD   time.Duration
	}{
		{
			"already dead",
			func(c control) { c.closeDone() },
			0,
		},
		{
			"responds within timeout",
			func(c control) {
				time.Sleep(defaultAsyncGracePeriod / 2)
				c.closeDone()
			},
			defaultAsyncGracePeriod / 2,
		},
		{
			"timeout",
			nil,
			defaultAsyncGracePeriod,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctrl := control{}

				c := newTestingComponent(t)
				c.doneCh, ctrl.closeDone = testutil.ChanWithCloser[struct{}](0)

				calledRunCtxCancel := false
				c.runCtxCancel = func() {
					calledRunCtxCancel = true
				}

				ctx, cancel := context.WithCancelCause(t.Context())
				defer cancel(errors.New("test done"))

				if tt.control != nil {
					go tt.control(ctrl)
					synctest.Wait()
				}

				t0 := time.Now()
				c.shutdownViaContext(ctx)
				test.Eq(t, tt.wantD, time.Since(t0))

				wantCalled := tt.name != "already dead" // So sue me...
				test.Eq(t, wantCalled, calledRunCtxCancel)
			})
		})
	}
}
