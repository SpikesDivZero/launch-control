// Package launch provides a way to launch and monitor components within an application.
//
// If any component exits, or if [RequestStop] is called, then the application shuts down so that it can be replaced
// by another instance. (App instance replacement is assumed to be provided externally -- e.g. k8s, systemd, etc.)
//
// Shutdown order is the reverse of the [Launch] order, as one would commonly expect.
package launch

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/spikesdivzero/launch-control/internal/controller"
)

// The bulk of the controller is implemented internally.
// We expose a documented struct here, as opposed to an interface, primarily for easier reading and godoc's sake.

// A Controller is the heart of the package. Components are Launched inside of the Controller, and the Controller
// provides a minimal interface to manage the application's lifecycle.
type Controller struct {
	impl *controller.Controller
}

func NewController(ctx context.Context) Controller {
	return Controller{impl: controller.New(ctx)}
}

// Launch builds a component from the provided name and options, then launches it inside of the controller.
// This blocks until the component launch has finished (regardless of success or failure).
//
// Required options: Nearly every option is, as the name suggests, optional. However you must provide exactly
// one of Run+Shutdown or Start+Stop as one of the options, as this defines how the component should execute.
//
// If a Launch request comes in after the controller has started shutting down, the request will be silently
// discarded.
func (c *Controller) Launch(name string, opts ...Options) {
	comp, err := buildComponent(name, opts...)
	if err != nil {
		panic(fmt.Sprintf("component build failed: %v", err))
	}
	c.impl.Launch(name, comp)
}

// RequestStop signals to the controller that it's time to exit, with an optional error explaining why.
//
// It's safe to call as multiple times. Only the first non-nil error is recorded.
func (c *Controller) RequestStop(reason error) {
	c.impl.RequestStop(reason)
}

// Wait blocks until the controller's internals exit, and then returns the result of [Err].
func (c *Controller) Wait() error {
	return c.impl.Wait()
}

// Err returns the first non-nil error recorded by the controller (including calls to [RequestStop]).
func (c *Controller) Err() error {
	return c.impl.Err()
}

func (c *Controller) AllErrors() []error {
	return c.impl.AllErrors()
}

// Sets a logger for the controller to use.
//
// Primarily intended for debugging, and the records emitted are not guaranteed to be useful.
func (c *Controller) SetLogger(log *slog.Logger) {
	if log == nil {
		panic("SetLogger: log must not be nil")
	}

	// FIXME: Should it be an error to set this after the first component is launched?
	c.impl.Log = log
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
func (c *Controller) SetInternalAsyncGracePeriod(d time.Duration) {
	if d <= 0 {
		panic("AsyncGracePeriod must be a positive, non-zero value")
	}
	if d > 5*time.Second {
		panic("AsyncGracePeriod must be reasonable") // it's a grace period, not a timeout...
	}

	c.impl.AsyncGracePeriod = d
}
