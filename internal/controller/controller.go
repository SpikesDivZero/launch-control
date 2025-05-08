package controller

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

type Component interface {
	ConnectController(
		log *slog.Logger,
		notifyOnExited func(error),
	)
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
	WaitReady(ctx context.Context, abortLoopCh <-chan struct{}) error
}

type Controller struct {
	ctx context.Context
	log *slog.Logger

	// Control Loop related bits.
	stateMu         sync.Mutex
	lifecycleState  lifecycleState
	doneCh          chan struct{}
	requestStopCh   chan struct{}
	requestLaunchCh chan launchRequest
	firstError      error
	components      []Component
}

func New(ctx context.Context, log *slog.Logger) *Controller {
	return &Controller{
		ctx: ctx,
		log: log,

		lifecycleState:  lifecycleNew,
		doneCh:          make(chan struct{}),
		requestStopCh:   make(chan struct{}),
		requestLaunchCh: make(chan launchRequest, 10), // reduce risk of deadlock
	}
}

func (c *Controller) Launch(name string, comp Component) {
	comp.ConnectController(c.log, func(err error) {
		if err != nil {
			err = fmt.Errorf("component %v run exited: %w", name, err)
		}
		c.RequestStop(err)
	})

	<-c.sendLaunchRequest(name, comp)
}

// Split out so that the lock boundary is clearly defined.
//
// We need the lock to write, but we do not want to be holding the lock while we're waiting for the request to finish.
//
// Aside, we return a bidirectional channel to make testing easier, but the caller should never close the returned chan.
func (c *Controller) sendLaunchRequest(name string, comp Component) chan struct{} {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	if c.lifecycleState == lifecycleNew {
		c.lifecycleState = lifecycleAlive
		go c.controlLoop()
	}

	doneCh := make(chan struct{})

	if c.lifecycleState != lifecycleAlive {
		close(doneCh)
		return doneCh
	}

	c.requestLaunchCh <- launchRequest{name, comp, doneCh}
	return doneCh
}

func (c *Controller) RequestStop(reason error) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	// RequestStop can be called multiple times.

	// We record the first error we see across all these calls, even if another stop request is already processed.
	if c.firstError == nil {
		c.firstError = reason
	}

	// We shouldn't panic on a second call.
	select {
	case <-c.requestStopCh:
		return // already closed
	default:
		close(c.requestStopCh)
	}

	// The only supported abnormal transition is New->Dead direct.
	// Normal state transition (Alive->Dying) is handled by the control loop.
	if c.lifecycleState == lifecycleNew {
		c.lifecycleState = lifecycleDead
		close(c.doneCh)
		close(c.requestLaunchCh)
	}
}

func (c *Controller) Wait() error {
	<-c.doneCh
	return c.Err()
}

func (c *Controller) Err() error {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	return c.firstError
}
