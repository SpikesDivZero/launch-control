package launch

import (
	"log/slog"
	"time"

	"github.com/spikesdivzero/launch-control/internal/controller"
)

type ControllerOption func(*controller.Controller)

// Sets a logger for the controller to use.
//
// Primarily intended for debugging, and the records emitted are not guaranteed to be useful.
func WithControllerLogger(log *slog.Logger) ControllerOption {
	if log == nil {
		panic(optionNilArgError{"WithControllerLogger", "log"})
	}

	return func(c *controller.Controller) {
		c.Log = log
	}
}

// Sets the async settlement/grace period for async operations. Default is 100ms.
//
// Deprecated: This is an internal detail, and probably shouldn't be used.
// Marked as deprecated so it'll be hidden by default.
//
// Go does not provide any guarantees about the order that coroutines are executed in, which case of a select will
// be selected for the wakeup condition, or a way to detect if a goroutine is scheduled for a wakeup.
//
// As such, we internally use a small grace period after some operations to
// give us the best chance to fully capture all outcomes, making the observable result more reliable (but still
// imperfect).
//
// If you're running on a slower or overloaded system, and are seeing results that are not quite stable, it may help
// to raise this to be a bit larger.
//
// However, do so at your own risk, since we don't advertise where this is used or how, and it's subject to change
// at any time.
//
// Primarily intended for debugging, and the records emitted are not guaranteed to be useful.
func WithControllerInternalAsyncGracePeriod(d time.Duration) ControllerOption {
	if d <= 0 {
		panic("AsyncGracePeriod must be a positive, non-zero value")
	}
	if d > 5*time.Second {
		panic("AsyncGracePeriod must be reasonable") // it's a grace period, not a timeout...
	}

	return func(c *controller.Controller) {
		c.AsyncGracePeriod = d
	}
}
