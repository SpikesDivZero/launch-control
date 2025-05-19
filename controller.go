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

	"github.com/spikesdivzero/launch-control/internal/controller"
)

// The bulk of the controller is implemented internally.
// We expose a documented struct here, as opposed to an interface, primarily for easier reading and godoc's sake.

// A Controller is the heart of the package. Components are Launched inside of the Controller, and the Controller
// provides a minimal interface to manage the application's lifecycle.
type Controller struct {
	impl *controller.Controller
}

func NewController(ctx context.Context, opts ...ControllerOption) Controller {
	c := Controller{impl: controller.New(ctx)}
	for _, opt := range opts {
		opt(c.impl)
	}
	return c
}

// Launch builds a component from the provided name and options, then launches it inside of the controller.
// This blocks until the component launch has finished (regardless of success or failure).
//
// Required options: Nearly every option is, as the name suggests, optional. However you must provide exactly
// one of [WithRun] or [WithStartStop] as one of the options, as this defines how the component should execute.
//
// If a Launch request comes in after the controller has started shutting down, the request will be silently
// discarded.
func (c *Controller) Launch(name string, opts ...ComponentOption) {
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
