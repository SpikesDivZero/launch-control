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

// Any errors returned by THIS method are a signal to waitReady_Loop to stop abort the main loop and
// instead return the error we provided.
//
// Returned errors from ImplCheckReady (or AsyncCall) are a policy matter as to whether or not we return them
// from CheckOnce. For now, the policy will be that any errors returned from AsyncCall (timeout) or ImplCheckReady
// are fatal to the application lifecycle.
//
// TODO: consider add a component option that lets us define the policy on a controller-wide and/or per-component basis
func (c *Component) waitReady_CheckOnce(ctx context.Context) (bool, error) {
	select {
	case <-c.doneCh:
		return false, lcerrors.ErrWaitReadyComponentExited
	default:
	}

	resultCh := AsyncCall(ctx, "CheckReady.CallTimeout", c.CheckReadyOptions.CallTimeout, c.asyncGracePeriod,
		func(ctx context.Context) Pair[bool, error] {
			r, err := c.ImplCheckReady(ctx)
			return Pair[bool, error]{r, err}
		})

	select {
	case result := <-resultCh:
		rp, callErr := result.Values()
		if callErr != nil {
			return false, callErr
		}
		return rp.Values()

	case <-c.doneCh:
		return false, lcerrors.ErrWaitReadyComponentExited
	}
}
