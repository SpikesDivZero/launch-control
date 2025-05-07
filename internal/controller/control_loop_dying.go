package controller

import "slices"

// The contents of this file run when lifecycleState is lifecycleDying.

func (c *Controller) controlLoop_Dying() {
	c.clAssertState("controlLoop_Dying", lifecycleDying)

	// All outstanding launch requests must be summarily discarded.
	close(c.requestLaunchCh)
	for req := range c.requestLaunchCh {
		close(req.doneCh)
	}

	// Run the graceful shutdown procedure (stopping all components in the reverse order of when they were started).
	for _, comp := range slices.Backward(c.components) {
		c.clDyingDoShutdown(comp)
	}
}

func (c *Controller) clDyingDoShutdown(comp Component) {
	_ = comp.Shutdown(c.ctx) // TODO: handle error
}
