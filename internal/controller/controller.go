package controller

import (
	"context"
	"log/slog"
	"slices"
	"sync"

	"github.com/spikesdivzero/launch-control/internal/component"
)

// If a function is undocumented, first check to see if it's documented in the public interface.

type Component interface {
	Register(context.Context, *slog.Logger, component.ControllerCallbacks)
	Start(context.Context)
	Stop()
}

type Controller struct {
	// We'll be locking so infrequently it shouldn't hurt to claim the entire struct.
	mu sync.Mutex

	ctx context.Context
	log *slog.Logger

	comps []Component
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

func (c *Controller) Launch(name string, comp Component) {
	var canLaunch bool
	c.inLock(func() {
		if c.state == runStateNew {
			c.state = runStateAlive
			go c.worker_main()
		}
		canLaunch = c.state == runStateAlive
	})

	if !canLaunch {
		return
	}

	req := launchRequest{
		Name:   name,
		Comp:   comp,
		DoneCh: make(chan struct{}),
	}
	c.requestLaunchCh <- req
	<-req.DoneCh
}

func (c *Controller) registerComponent(name string, comp Component) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.comps = append(c.comps, comp)

	comp.Register(c.ctx, c.log, c.makeControllerCallbacksFor(name, comp))
}

func (c *Controller) makeControllerCallbacksFor(name string, comp Component) component.ControllerCallbacks {
	_, _ = name, comp // I may use this in the future. For now, pass.

	return component.ControllerCallbacks{
		RequestStop: func(reason error) {
			c.RequestStop(reason)
		},
		ComponentError: func(err error) {
			c.recordError(err, true)
		},
	}
}

func (c *Controller) RequestStop(reason error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.recordError(reason, false)

	// Shortcut: New->Dead is a special case
	if c.state == runStateNew {
		c.state = runStateDead
		close(c.requestLaunchCh)
		close(c.requestStopCh)
		close(c.deadCh)
		return
	}

	// In all other cases, we close the channel if it's not already closed.
	// The main worker is then responsible for changing our state into Dying.
	select {
	case <-c.requestStopCh:
	default:
		close(c.requestStopCh)
	}
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
