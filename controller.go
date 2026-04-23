package launchcontrol

import (
	"context"
	"fmt"
	"log/slog"
)

// Controller is the primary interface into this package.
//
// The Controller tracks the lifecycle of all components spawned within it, and offers a way to
// block until all components have exited.
type Controller struct {
}

// New returns a new Controller, using the provided context as it's root context.
//
// Any contexts passed into a component will be children of this context.
func New(ctx context.Context) *Controller {
	return &Controller{} // TODO
}

// Launch builds and then runs/starts a component, using the provided Options.
//
// At least one Options must be provided. The provided Options are all merged down into a
// single set of Options. The final Options provided ot this call must provide Run or Start
// function, along with a Stop function.
//
// In the that the provided Options fail validation, the Launch call will panic.
//
// If the controller is shutting down or dead, then the Launch request will be ignored.
func (c *Controller) Launch(name string, opts ...Options) {
	if name == "" {
		panic("Controller.Launch: must provide a name")
	}
	if len(opts) == 0 {
		panic("Controller.Launch: must pass at least one Options")
	}

	merged, err := mergeOptions(opts)
	if err != nil {
		panic(fmt.Sprintf("Controller.Launch: option validation failed: %v", err))
	}

	if err := merged.validate(); err != nil {
		panic(fmt.Sprintf("Controller.Launch: option validation failed: %v", err))
	}

	panic("NYI: Controller.Launch")
}

// RequestStop asks that the controller begin the shutdown process, with the provided reason.
//
// If the reason is not nil, then it's recorded as an error, which can returned by
// [Controller.Err] or [Controller.AllErrors].
func (c *Controller) RequestStop(reason error) {
	panic("NYI: Controller.RequestStop")
}

// Wait blocks until all components have finished shutting down.
//
// It returns the value from [Controller.Err] for convenience.
//
// If Wait is called before any components have been launched, then the call panics.
func (c *Controller) Wait() error {
	panic("NYI: Controller.Wait")
}

// Err returns the first non-nill error observed by the controller.
//
// In the event that no errors have been observed, this returns nil.
func (c *Controller) Err() error {
	panic("NYI: Controller.Err")
}

// AllErrors returns a slice containing all non-nill errors observed by the controller.
func (c *Controller) AllErrors() []error {
	panic("NYI: Controller.AllErrors")
}

// SetLogger allows you to enable logging for this package.
//
// The default logger used by this package discards all records.
func (c *Controller) SetLogger(log *slog.Logger) {
	panic("NYI: Controller.SetLogger")
}
