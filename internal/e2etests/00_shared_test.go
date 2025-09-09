package e2etests

import (
	"context"
	"testing"

	"github.com/spikesdivzero/launch-control"
)

// For many of these tests, I'm going to be wrapping them in synctest.Test mainly for the benefit of deadlock detection.
// Also, if any timings concerns do come into play, we won't end up having tests run for hours.

func newController(t *testing.T) launch.Controller {
	return launch.NewController(t.Context())
}

func withDummyStartStop() launch.ComponentOption {
	return launch.WithStartStop(
		func(context.Context) error { return nil },
		func(context.Context) error { return nil })
}
