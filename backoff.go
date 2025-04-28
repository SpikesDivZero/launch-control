package launch

import (
	"math"
	"math/rand/v2"
	"time"
)

type BackoffFunc func() time.Duration

func ConstBackoff(delay time.Duration) BackoffFunc {
	return func() time.Duration {
		return delay
	}
}

func ExpBackoff(minDelay, maxDelay time.Duration, exp float64, jitter bool) BackoffFunc {
	minDelayF := float64(minDelay)
	maxDelayF := float64(maxDelay)

	if minDelayF == 0 || maxDelayF == 0 {
		panic("ExpBackoff: min and max delay must not be zero")
	}
	if minDelay > maxDelay {
		panic("ExpBackoff: minDelay must be less than maxDelay")
	}

	attempt := 0
	return func() time.Duration {
		delayF := minDelayF * math.Pow(exp, float64(attempt))
		attempt++

		if jitter {
			jv := 0.9 + rand.Float64()*0.2 // uniform +/- 10%
			delayF *= jv
		}

		delayF = min(delayF, maxDelayF)
		delayF = max(delayF, minDelayF)
		return time.Duration(delayF)
	}
}
