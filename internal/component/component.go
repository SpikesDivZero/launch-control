package component

import (
	"context"
	"log/slog"
	"time"
)

const AsyncGracePeriod = 100 * time.Millisecond

type ControllerCallbacks struct {
	// Requests that the controller enter a shutting down state (if it's not already doing so)
	RequestStop func(c *Component, reason error)

	// Notifies the controller of an error, that doesn't necessarily require transitioning into
	// a shutting down state. (Or, in the case of ImplStop, we're already stopping anyways)
	ComponentError func(c *Component, err error)
}

// For now, a thin wrapper around the stuff we import from the public interface Options.
type Component struct {
	Name string

	ImplRun               func(context.Context) error
	ImplCheckReady        func(context.Context) (bool, error)
	ImplCheckReadyBackoff func() time.Duration
	ImplStop              func(context.Context) error

	// We won't use this directly, but we keep a pointer to it here for other test inspection.
	SSW *StartStopWrapper

	log       *slog.Logger
	callbacks ControllerCallbacks

	ctx       context.Context
	ctxCancel context.CancelCauseFunc

	runCtx       context.Context
	runCtxCancel context.CancelCauseFunc
	runExitedCh  <-chan struct{}
}

func (c *Component) Register(ctx context.Context, log *slog.Logger, callbacks ControllerCallbacks) {
	c.ctx, c.ctxCancel = context.WithCancelCause(ctx)
	c.runCtx, c.runCtxCancel = context.WithCancelCause(c.ctx)

	c.log = log
	c.callbacks = callbacks
}

func (c *Component) Start() {
	exitedCh := make(chan struct{})
	resultCh := make(chan error, 1)

	c.runExitedCh = exitedCh

	go func() {
		defer close(exitedCh)
		defer close(resultCh)

		resultCh <- c.ImplRun(c.runCtx)
	}()

	go c.monitorExit(resultCh)

	c.waitReady()
}

// Designed to run in a separate goroutine, this monitors the result channel provided by Start.
// When ImplRun returns, we get a message containing the resultant error (or nil) which we'll
// then pass up to the Controller.
//
// LEAKY GOROUTINE: This will leak by design. Practically speaking, this leak occurs when the
// application is shutting down, and the user-provided Run function
func (c *Component) monitorExit(resultCh chan error) {
	err, ok := <-resultCh
	err = CheckPrematureChannelClose("resultCh", err, ok)

	// NOTE: Do not cancel the root context here, since ImplStop may still be alive.
	//
	// We do not know which of ImplRun and ImplStop will take the longest to finish, and
	// the stop process is only considered succssfully completed when both have finished.

	c.callbacks.RequestStop(c, WrapComponentError(c, "exited", err))
}

func (c *Component) Stop() {
	// For our stop process, we'll define it as blocking until _both_ of ImplStop and ImplRun have returned.
	//
	// For now, we'll take the simplest approach possible. We still want to implement proper timeouts, so
	// that's a thing to address later on.

	err := c.ImplStop(c.ctx)
	c.callbacks.ComponentError(c, WrapComponentError(c, "stop", err))

	<-c.runExitedCh
}
