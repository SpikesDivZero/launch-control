package component

import (
	"context"
	"time"
)

func (c *Component) waitReady() {
	if c.ImplCheckReady == nil {
		return
	}

	if c.ImplCheckReadyBackoff == nil {
		c.ImplCheckReadyBackoff = func() time.Duration { return 0 }
	}

	// The wait process should happen inside of the run context, as we want to stop waiting if
	// a request to stop running comes in, and runCtx may close sooner than the component base
	// context otherwise would.
	c.waitReady_loop(
		c.runCtx,
		c.ImplCheckReady,
		c.ImplCheckReadyBackoff,
		c.callbacks.RequestStop,
	)
}

// This is pulled out into it's own function as otherwise the main loop is kind of a pain to test.
func (c *Component) waitReady_loop(
	ctx context.Context,
	fnCheck func(context.Context) (bool, error),
	fnBackoff func() time.Duration,
	fnRequestStop func(*Component, error),
) bool {
	// We should abort when our main context is done, or run has already exited.
	// In both cases, there's no point in continuing our loop
	abortCh, shouldAbort := c.waitReady_createShouldAbort(ctx)

	for {
		if shouldAbort() {
			return false
		}

		ready, err := fnCheck(ctx)
		if err != nil {
			// User-provided error, so we won't wrap it
			fnRequestStop(c, err)
			return false
		} else if ready {
			return true
		}

		if shouldAbort() {
			return false
		}

		c.waitReady_sleep(fnBackoff(), abortCh)
	}
}

// The returned channel is closed when the loop should be aborted.
//
// The returned function returns true when the channel is closed.
func (c *Component) waitReady_createShouldAbort(ctx context.Context) (<-chan struct{}, func() bool) {
	abortCh := make(chan struct{})

	// Before we start any async work, let's first check to see if we're already aborting.
	//
	// This is necessary to address the edge case when either condition is met when we enter the function,
	// and the monitoring coroutine won't run for the first time up for a while yet to come. (Potential race)
	alreadyAborting := false
	select {
	case <-ctx.Done():
		alreadyAborting = true
	case <-c.runExitedCh:
		alreadyAborting = true
	default:
	}
	if alreadyAborting {
		close(abortCh)
		return abortCh, func() bool { return true }
	}

	go func() {
		select {
		case <-ctx.Done():
		case <-c.runExitedCh:
		}
		close(abortCh)
	}()

	return abortCh, func() bool {
		select {
		case <-abortCh:
			return true
		default:
			return false
		}
	}
}

func (c *Component) waitReady_sleep(d time.Duration, abortCh <-chan struct{}) {
	if d <= 0 {
		return
	}

	select {
	case <-time.After(d):
	case <-abortCh:
	}
}
