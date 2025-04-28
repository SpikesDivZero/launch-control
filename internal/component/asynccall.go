package component

import (
	"context"
	"errors"
	"time"
)

var errPrematureChannelClose = errors.New("channel closed without sending a result value")

type Pair[T1, T2 any] struct {
	a T1
	b T2
}

func (p Pair[T1, T2]) Values() (T1, T2) {
	return p.a, p.b
}

// Wraps a call to the provided function, returning it's result in a channel.
// In the event of a timeout, the context.Cause value is returned in
func AsyncCall[RT any](
	ctx context.Context,
	timeout time.Duration,
	f func(context.Context) RT,
	timeoutGrace time.Duration,
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

	ctx, ctxCancel := context.WithTimeout(ctx, timeout)
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

		writeUserResult := func(r RT, ok bool) {
			if ok {
				returnCh <- ReturnType{r, nil}
			} else {
				// Probably shouldn't happen, but just in case
				returnCh <- ReturnType{zeroRT, errPrematureChannelClose}
			}
		}

		// Wait for either the call to complete or our call context to be cancelled.
		select {
		case r, ok := <-innerCh:
			writeUserResult(r, ok)
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
			writeUserResult(r, ok)
		}
	}()

	return returnCh
}
