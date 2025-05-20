package launch

import (
	"log/slog"
	"testing"
	"time"

	"github.com/shoenig/test"
	"github.com/spikesdivzero/launch-control/internal/controller"
	"github.com/spikesdivzero/launch-control/internal/testutil"
)

func TestWithControllerLogger(t *testing.T) {
	c := controller.New(t.Context())

	log := slog.New(slog.Default().Handler())
	WithControllerLogger(log)(c)
	test.Eq(t, log, c.Log)

	t.Run("panics on nil", func(t *testing.T) {
		defer testutil.WantPanic(t, optionNilArgError{"WithControllerLogger", "log"}.Error())
		WithControllerLogger(nil)(c)
	})
}

func TestWithControllerInternalAsyncGracePeriod(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		c := controller.New(t.Context())

		WithControllerInternalAsyncGracePeriod(744 * time.Millisecond)(c)
		test.Eq(t, 744*time.Millisecond, c.AsyncGracePeriod)
	})

	t.Run("panic on zero", func(t *testing.T) {
		defer testutil.WantPanic(t, "AsyncGracePeriod must be a positive, non-zero value")
		WithControllerInternalAsyncGracePeriod(0)
	})

	t.Run("panic on negative", func(t *testing.T) {
		defer testutil.WantPanic(t, "AsyncGracePeriod must be a positive, non-zero value")
		WithControllerInternalAsyncGracePeriod(-1)
	})

	t.Run("panic on unreasonable", func(t *testing.T) {
		defer testutil.WantPanic(t, "AsyncGracePeriod must be reasonable")
		WithControllerInternalAsyncGracePeriod(time.Minute)
	})
}
