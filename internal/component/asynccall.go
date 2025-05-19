package component

import (
	"context"
	"time"

	"github.com/spikesdivzero/launch-control/internal/lcerrors"
)

type Pair[T1, T2 any] struct {
	a T1
	b T2
}

func (p Pair[T1, T2]) Values() (T1, T2) {
	return p.a, p.b
}

// Wraps a call to the provided function, returning it's result in a channel.
//
// If the error slot contains [errPrematureChannelClose], it means the provided function invoked [runtime.Goexit].
//
// Any other non-nil value in the error slot will be a return from [context.Cause]. This includes a timeout being
// hit, as we do not differentiate a [context.DeadlineExceeded] as being from this timeout or a parent timeout.
func AsyncCall[RT any](
	ctx context.Context,
	timeoutSource string,
	timeout time.Duration,
	timeoutGrace time.Duration,
	f func(context.Context) RT,
) <-chan Pair[RT, error] {

	// Our top-level return type and channel.
	type ReturnType = Pair[RT, error]
	returnCh := make(chan ReturnType, 1)
	var zeroRT RT

	// Don't call anything if we're already dead.
	if ctx.Err() != nil {
		returnCh <- ReturnType{zeroRT, context.Cause(ctx)}
		close(returnCh)
		return returnCh
	}

	ctx, ctxCancel := context.WithTimeoutCause(ctx, timeout, lcerrors.ContextTimeoutError{Source: timeoutSource})
	// We don't cancel our ctx here, but instead inside our monitoring goroutine

	// Our inner call and result channel
	innerCh := make(chan RT, 1)
	go func() {
		defer close(innerCh)
		innerCh <- f(ctx)
	}()

	go func() {
		defer close(returnCh)
		defer ctxCancel()

		// Wait for either the call to complete or our call context to be cancelled.
		select {
		case r, ok := <-innerCh:
			returnCh <- ReturnType{r, checkForPrematureClose(nil, ok)}
			return

		case <-ctx.Done():
			// Fall through
		}

		// The call context has been cancelled.

		// If there's no grace period, then we're done.
		if timeoutGrace == 0 {
			returnCh <- ReturnType{zeroRT, context.Cause(ctx)}
			return
		}

		// Okay, we have a grace period. Even if the parent context has been cancelled, we'll still honor it
		// just in case the message it returns is useful for debugging.
		//
		// NOTE: This means the caller should always use a tiny grace period, but since it's an internal-only
		// function, I'm OK with this tradeoff.
		select {
		case <-time.After(timeoutGrace):
			returnCh <- ReturnType{zeroRT, context.Cause(ctx)}

		case r, ok := <-innerCh:
			returnCh <- ReturnType{r, checkForPrematureClose(nil, ok)}
		}
	}()

	return returnCh
}
