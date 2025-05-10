package component

import (
	"context"
	"time"

	"github.com/spikesdivzero/launch-control/internal/lcerrors"
)

func (c *Component) Shutdown(ctx context.Context) error {
	// Stage 1: Prefer a normal shutdown via user-provided ImplShutdown
	// Stage 2: If that fails, attempt a shutdown via context cancellation.
	// Stage 3: Give up and abandon the component, so as to not block the app shutdown.
	//
	// Each Via function is responsible for checking isDead at the start, in order to keep
	// testing the main Shutdown() function as simple as possible.
	c.shutdownViaImpl(ctx)
	c.shutdownViaContext(ctx)
	if c.isDead() {
		return nil
	} else {
		return lcerrors.ErrShutdownAbandonedNonResponsive
	}
}

func (c *Component) isDead() bool {
	select {
	case <-c.doneCh:
		return true
	default:
		return false
	}
}

// Returns nil on success. Error is just for internal test validations.
func (c *Component) shutdownViaImpl(ctx context.Context) {
	if c.isDead() {
		return
	}

	ctx, ctxCancel := context.WithTimeout(ctx, c.ShutdownOptions.CompletionTimeout)
	defer ctxCancel()

	// We need to wait for BOTH ImplShutdown to complete, as well as ImplRun to return.
	//
	// Why? As a concrete example, take a look at net/http.Shutdown. There, Server.Shutdown causes
	// Server.Serve to return almost immediately, while Shutdown blocks until the graceful shutdown
	// process has completed.
	//
	// Rather than over-complicate things, this method will focus on calling ImplShutdown and
	// waiting on it to return. Only after that happens will it check/wait on ImplRun being done.

	// FIXME: use a call grace timeout provided by controller, instead of const 100ms?
	resultCh := AsyncCall(ctx, c.ShutdownOptions.CallTimeout, 100*time.Millisecond, c.ImplShutdown)
	if userErr, callErr := (<-resultCh).Values(); callErr != nil {
		c.logError("shutdown (impl)", callErr)
	} else if userErr != nil {
		c.logError("shutdown (impl)", userErr)
	}

	select {
	case <-ctx.Done():
		// CompletionTimeout expired.
		c.logError("shutdown (impl)", context.Cause(ctx))
	case <-c.doneCh:
		// ImplRun finished.
	}
}

// Returns nil on success. Error is just for internal test validations.
func (c *Component) shutdownViaContext(context.Context) {
	if c.isDead() {
		return
	}

	c.runCtxCancel()

	// FIXME: Should we use use ctx? For now, we have a fixed timeout of 100ms.
	select {
	case <-c.doneCh:
		// Responded successfully, and is now exited
	case <-time.After(100 * time.Millisecond): // FIXME: time provided by controller?
		// Did not respond, and is still alive
	}
}
