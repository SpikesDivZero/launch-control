package component

import (
	"context"
	"fmt"
	"time"
)

func (c *Component) Start(ctx context.Context) error {
	if c.doneCh != nil {
		panic("Start called twice?")
	}
	doneCh := make(chan struct{})
	c.doneCh = doneCh

	// The runCtx should only be used for the ImplRun call.
	// All other cases in here should continue to use the parent context.
	var runCtx context.Context
	runCtx, c.runCtxCancel = context.WithCancel(ctx)

	runErrCh := make(chan error, 1)
	go func() {
		defer func() {
			close(runErrCh)
			close(doneCh)
		}()
		runErrCh <- c.ImplRun(runCtx)
	}()

	go c.monitorExit(ctx, runErrCh)

	if err := c.waitReady(ctx); err != nil {
		return fmt.Errorf("failed to become ready: %w", err)
	}
	return nil
}

func (c *Component) monitorExit(ctx context.Context, runErrCh <-chan error) {
	select {
	case <-ctx.Done():
		// Fall through

	case err, ok := <-runErrCh:
		c.notifyOnExited(checkForPrematureClose(err, ok))
		return
	}

	select {
	case <-time.After(100 * time.Millisecond): // FIXME: dynamic value, provided by controller?
		c.log.Debug("Component.monitorExit is exiting while the component is still alive.")

	case err, ok := <-runErrCh:
		c.notifyOnExited(checkForPrematureClose(err, ok))
		return
	}
}
