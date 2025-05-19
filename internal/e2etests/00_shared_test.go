package e2etests

import (
	"testing"

	"github.com/spikesdivzero/launch-control"
)

// For many of these tests, I'm going to be wrapping them in synctest.Run mainly for the benefit of deadlock detection.
// Also, if any timings concerns do come into play, we won't end up having tests run for hours.

func newController(t *testing.T) launch.Controller {
	return launch.NewController(t.Context())
}
