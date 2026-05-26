package controller

import (
	"context"
	"fmt"
	"slices"
	"time"
)

// The async (worker) components of the controller.

type launchRequest struct {
	Name   string
	Comp   Component
	DoneCh chan struct{}
}

// The entrypoint for our core worker's coroutine.
func (c *Controller) worker_main() {
	c.worker_changeState(runStateAlive, runStateAlive)
	c.worker_alive()

	c.worker_changeState(runStateAlive, runStateDying)
	c.worker_dying()

	c.worker_changeState(runStateDying, runStateDead)
	close(c.deadCh)
}

func (c *Controller) worker_changeState(from, to runState) {
	c.inLock(func() {
		if c.state != from {
			panic(fmt.Errorf("Controller worker state mismatch; got %s != expected %s", c.state, from))
		}

		c.state = to
	})
}

func (c *Controller) worker_alive() {
	for {
		// Wait for either a job to come in, or a request to stop.
		select {
		case <-c.requestStopCh:
			return
		case req := <-c.requestLaunchCh:
			c.worker_doLaunch(req)
		}
	}
}

func (c *Controller) worker_doLaunch(req launchRequest) {
	defer close(req.DoneCh)

	// If a stop has been requested, then that takes priority over launching something new.
	select {
	case <-c.requestStopCh:
		return
	default:
	}

	// We want a launch context so we can interrupt the wait-ready loop
	startCtx, startCtxCancel := context.WithCancel(c.ctx)
	defer startCtxCancel()

	// Monitor the requestStop channel to interrupt our launch context.
	go func() {
		select {
		case <-c.requestStopCh:
		case <-startCtx.Done():
		}
		startCtxCancel()
	}()

	c.registerComponent(req.Name, req.Comp)

	req.Comp.Start(startCtx)
}

func (c *Controller) worker_dying() {
	go c.worker_discardLaunchRequests()

	for _, comp := range slices.Backward(c.comps) {
		comp.Stop()
	}
}

// LEAKY GOROUTINE. After we transition into the dying state, we run this for at least 5 seconds to purge any
// launch requests that slip through the cracks (simpler than doing the right thing)
//
// At the end of this process, we close the launch request channel.
func (c *Controller) worker_discardLaunchRequests() {
	defer close(c.requestLaunchCh)

	for {
		select {
		case <-time.After(5 * time.Second):
			return
		case req := <-c.requestLaunchCh:
			close(req.DoneCh)
		}
	}
}
