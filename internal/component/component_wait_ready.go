package component

import (
	"context"
	"time"

	"github.com/spikesdivzero/launch-control/internal/lcerrors"
)

func (c *Component) WaitReady(ctx context.Context, abortCh <-chan struct{}) error {
	if c.ImplCheckReady == nil {
		return nil
	}

	return waitReady_MainLoop(
		ctx,
		abortCh,
		c.CheckReadyOptions.MaxAttempts,
		c.waitReady_CheckOnce,
		c.waitReady_Backoff,
	)
}

func waitReady_MainLoop(
	ctx context.Context,
	abortCh <-chan struct{},
	maxAttempts int,
	checkOnce func(context.Context) (bool, error),
	backoff func(context.Context, <-chan struct{}) error,
) error {
	shouldAbort := func() error {
		select {
		case <-abortCh:
			return lcerrors.ErrWaitReadyAbortChClosed
		default:
			return nil
		}
	}

	for attempt := range maxAttempts {
		if err := shouldAbort(); err != nil {
			return err
		}

		if attempt > 0 {
			if err := backoff(ctx, abortCh); err != nil {
				return err
			}
			if err := shouldAbort(); err != nil {
				return err
			}
		}

		if ready, err := checkOnce(ctx); ready {
			return nil
		} else if err != nil {
			return err
		}
	}
	return lcerrors.ErrWaitReadyExceededMaxAttempts
}

func (c *Component) waitReady_Backoff(ctx context.Context, abortCh <-chan struct{}) error {
	d := c.CheckReadyOptions.Backoff()
	if d <= 0 {
		return nil
	}

	select {
	case <-time.After(d):
		return nil
	case <-abortCh:
		return lcerrors.ErrWaitReadyAbortChClosed
	case <-ctx.Done():
		return context.Cause(ctx)
	case <-c.doneCh:
		return lcerrors.ErrWaitReadyComponentExited
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
		return false, lcerrors.ErrWaitReadyComponentExited
	default:
	}

	resultCh := AsyncCall(ctx, "CheckReady.CallTimeout", c.CheckReadyOptions.CallTimeout, 100*time.Millisecond,
		func(ctx context.Context) bool {
			ready, err := c.ImplCheckReady(ctx)
			if ready {
				return true
			}
			if err != nil {
				c.logError("wait-ready", err)
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
		return false, lcerrors.ErrWaitReadyComponentExited
	}
}
