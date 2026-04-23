package launchcontrol

import "time"

type BackoffFunc func() time.Duration

// ConstBackoff returns a backoff function that always returns the same value.
func ConstBackoff(delay time.Duration) BackoffFunc {
	panic("NYI: ConstBackoff")
}

// ExpBackoff returns a backoff function that crudely approximates an exponential backoff.
//
// The approximate formula is "minDelay * pow(exp, attempt-1)".
// If enabled, Jitter is a random +/- 10% of the computed delay.
// As a final step, the calculated backoff is clamped to within [minDelay, maxDelay].
func ExpBackoff(minDelay, maxDelay time.Duration, exp float64, jitter bool) BackoffFunc {
	panic("NYI: ExpBackoff")
}
