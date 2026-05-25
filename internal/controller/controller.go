package controller

import (
	"context"
	"log/slog"
	"slices"
	"sync"

	"github.com/spikesdivzero/launch-control/internal/component"
)

// If a function is undocumented, first check to see if it's documented in the public interface.

type launchRequest struct{} // FIXME: Placeholder.

func init() {
	_ = (&Controller{}).inLock // FIXME: Placeholder.
}

type Controller struct {
	// We'll be locking so infrequently it shouldn't hurt to claim the entire struct.
	mu sync.Mutex

	ctx context.Context
	log *slog.Logger

	comps []*component.Component
	errs  []error

	state           runState
	requestLaunchCh chan launchRequest // Write launch requests to this channel
	requestStopCh   chan struct{}      // Closed when RequestStop is called for the first time
	deadCh          chan struct{}      // Closed when the controller enters the Dead state
}

func NewController(ctx context.Context) *Controller {
	return &Controller{
		ctx: ctx,
		log: slog.New(slog.DiscardHandler),

		state:           runStateNew,
		requestLaunchCh: make(chan launchRequest),
		requestStopCh:   make(chan struct{}),
		deadCh:          make(chan struct{}),
	}
}

func (c *Controller) inLock(f func()) {
	c.mu.Lock()
	defer c.mu.Unlock()

	f()
}

func (c *Controller) recordError(err error, acquireLock bool) {
	if err == nil {
		return
	}

	if acquireLock {
		c.mu.Lock()
		defer c.mu.Unlock()
	}

	c.errs = append(c.errs, err)
}

func (c *Controller) Launch(name string, comp *component.Component) {
	_ = c.comps // FIXME: Placeholder.
	panic("NYI: Controller.Launch")
}

func (c *Controller) RequestStop(reason error) {
	panic("NYI: Controller.RequestStop")
}

func (c *Controller) Wait() {
	<-c.deadCh
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
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state != runStateNew {
		panic("Controller.SetLogger is forbidden after the first Launch/RequestStop call")
	}

	c.log = log
}
