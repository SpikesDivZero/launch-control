package component

import (
	"context"
	"errors"
	"time"
)

var errWaitReadyComponentExited = errors.New("during waitReady: component exited")

var errWaitReadyExceededMaxAttempts = errors.New("component did not become ready within MaxAttempts")

func (c *Component) WaitReady(ctx context.Context) error {
	if c.ImplCheckReady == nil {
		return nil
	}

	return waitReady_MainLoop(
		ctx,
		c.CheckReadyOptions.MaxAttempts,
		c.waitReady_CheckOnce,
		c.waitReady_Backoff,
	)
}

func waitReady_MainLoop(
	ctx context.Context,
	maxAttempts int,
	checkOnce func(context.Context) (bool, error),
	backoff func(context.Context) error,
) error {
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
}

func (c *Component) waitReady_Backoff(ctx context.Context) error {
	d := c.CheckReadyOptions.Backoff()
	if d <= 0 {
		return nil
	}

	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	case <-c.doneCh:
		return errWaitReadyComponentExited
	}
}

// Nuance:
//
// Any errors returned by THIS method are a signal to waitReady_Loop to stop abort the main loop and
// instead return the error we provided.
//
// If the User-provided ImplCheckReady returns an error, that's something we need to log and then retry.
func (c *Component) waitReady_CheckOnce(ctx context.Context) (bool, error) {
	select {
	case <-c.doneCh:
		return false, errWaitReadyComponentExited
	default:
	}

	resultCh := AsyncCall(ctx, c.CheckReadyOptions.CallTimeout, 100*time.Millisecond, func(ctx context.Context) bool {
		ready, err := c.ImplCheckReady(ctx)
		if ready {
			return true
		}
		if err != nil {
			c.logError("WaitReady", err)
		}
		return false
	})

	select {
	case result := <-resultCh:
		ready, callErr := result.Values()
		if callErr != nil {
			return false, callErr
		}
		return ready, nil

	case <-c.doneCh:
		return false, errWaitReadyComponentExited
	}
}
