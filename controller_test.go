package launch

import (
	"log/slog"
	"testing"
	"time"

	"github.com/shoenig/test"
	"github.com/spikesdivzero/launch-control/internal/testutil"
)

func TestController_SetLogger(t *testing.T) {
	c := NewController(t.Context())
	// Controller impl starts with a non-nil controller (using DiscardHandler)

	log := slog.New(slog.Default().Handler())
	c.SetLogger(log)
	test.Eq(t, log, c.impl.Log)

	t.Run("panics on nil", func(t *testing.T) {
		defer testutil.WantPanic(t, "SetLogger: log must not be nil")
		c.SetLogger(nil)
	})
}

func TestController_SetInternalAsyncGracePeriod(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		c := NewController(t.Context())
		c.SetInternalAsyncGracePeriod(744 * time.Millisecond)
		test.Eq(t, 744*time.Millisecond, c.impl.AsyncGracePeriod)
	})

	t.Run("panic on zero", func(t *testing.T) {
		defer testutil.WantPanic(t, "AsyncGracePeriod must be a positive, non-zero value")

		c := NewController(t.Context())
		c.SetInternalAsyncGracePeriod(0)
	})

	t.Run("panic on negative", func(t *testing.T) {
		defer testutil.WantPanic(t, "AsyncGracePeriod must be a positive, non-zero value")

		c := NewController(t.Context())
		c.SetInternalAsyncGracePeriod(-1)
	})

	t.Run("panic on unreasonable", func(t *testing.T) {
		defer testutil.WantPanic(t, "AsyncGracePeriod must be reasonable")

		c := NewController(t.Context())
		c.SetInternalAsyncGracePeriod(time.Minute)
	})
}
