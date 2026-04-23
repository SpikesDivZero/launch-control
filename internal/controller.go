package internal

import (
	"context"
	"log/slog"
	"slices"
	"sync"
)

// If a function is undocumented, first check to see if it's documented in the public interface.

type Controller struct {
	// We'll be locking so infrequently it shouldn't hurt to claim the entire struct.
	mu sync.Mutex

	ctx context.Context
	log *slog.Logger

	errs []error
}

func NewController(ctx context.Context) *Controller {
	return &Controller{
		ctx: ctx,
		log: slog.New(slog.DiscardHandler),
	}
}

func (c *Controller) Launch(name string, comp *Component) {
	panic("NYI: Controller.Launch")
}

func (c *Controller) RequestStop(reason error) {
	panic("NYI: Controller.RequestStop")
}

func (c *Controller) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.errs) > 0 {
		return c.errs[0]
	}
	return nil
}

func (c *Controller) AllErrors() []error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return slices.Clone(c.errs)
}

func (c *Controller) SetLogger(log *slog.Logger) {
	// TODO: If any components are already started, reject the request to change the logger.
	// This may be safe to wire through later on, but I don't want to deal with that.

	c.log = log
}
