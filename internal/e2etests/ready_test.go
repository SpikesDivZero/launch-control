package e2etests

import (
	"context"
	"errors"
	"math/rand/v2"
	"testing"
	"testing/synctest"
	"time"

	"github.com/shoenig/test"
	"github.com/spikesdivzero/launch-control"
	"github.com/spikesdivzero/launch-control/internal/lcerrors"
)

func TestReadyReturnsSuccess(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := newController(t)

		numCalls := 0
		checkReady := func(context.Context) (bool, error) {
			numCalls++
			switch numCalls {
			default:
				return false, nil
			case 4:
				return true, nil
			case 5:
				panic("should've stopped calling checkReady")
			}
		}

		c.Launch("test", withDummyStartStop(),
			launch.WithCheckReady(checkReady))

		time.AfterFunc(time.Second, func() { c.RequestStop(nil) })

		test.NoError(t, c.Wait())
		test.Eq(t, 4, numCalls)
	})
}

func TestReadyReturnsError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := newController(t)

		testErr := errors.New("fancy feast")
		numCalls := 0
		checkReady := func(context.Context) (bool, error) {
			if numCalls++; numCalls > 1 {
				panic("should've only been called once")
			}
			return true, testErr // error > status
		}

		c.Launch("test", withDummyStartStop(),
			launch.WithCheckReady(checkReady))
		test.ErrorIs(t, c.Wait(), lcerrors.ComponentError{Name: "test", Stage: "wait-ready", Err: testErr})
	})
}

func TestReadyMaxAttempts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := newController(t)

		numCalls := 0
		checkReady := func(context.Context) (bool, error) {
			if numCalls++; numCalls > 3 {
				panic("should've only been called 3 times")
			}
			return false, nil
		}

		c.Launch("test", withDummyStartStop(),
			launch.WithCheckReady(checkReady),
			launch.WithCheckReadyMaxAttempts(3))

		test.ErrorIs(t, c.Wait(), lcerrors.ComponentError{
			Name:  "test",
			Stage: "wait-ready",
			Err:   lcerrors.ErrWaitReadyExceededMaxAttempts,
		})
	})
}

func TestReadyBackoff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := newController(t)

		// synctest mocks time, so we want it at 0 to for the first check.
		// checkReady sets it to -1 after validating it
		var wantBackoff time.Duration
		numBackoff := 0
		lastCheck := time.Now() // to get a Since()==0 on first check

		backoff := func() time.Duration {
			if wantBackoff != -1 {
				panic("checkReady didn't validate the timeout -- double backoff call?")
			}
			wantBackoff = time.Second + time.Millisecond*time.Duration(rand.IntN(1000))
			numBackoff++
			return wantBackoff
		}

		checkReady := func(context.Context) (bool, error) {
			if wantBackoff == -1 {
				panic("checkReady called again without calling backoff?")
			}

			test.Eq(t, wantBackoff, time.Since(lastCheck))
			wantBackoff, lastCheck = -1, time.Now()
			return false, nil
		}

		c.Launch("test", withDummyStartStop(),
			launch.WithCheckReady(checkReady),
			launch.WithCheckReadyBackoff(backoff),
			launch.WithCheckReadyMaxAttempts(5))

		test.ErrorIs(t, c.Wait(), lcerrors.ComponentError{
			Name:  "test",
			Stage: "wait-ready",
			Err:   lcerrors.ErrWaitReadyExceededMaxAttempts,
		})
		test.Eq(t, 4, numBackoff)
	})
}
