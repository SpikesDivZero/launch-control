package controller

// The contents of this file run when lifecycleState is lifecycleAlive.
//
// I've split it into different files based on stages both for consistency with the component code,
// as well as clarity in the event that I need to extend this. (For instance, I may in the future
// add in the ability to launch a group of components concurrently.)

func (c *Controller) controlLoop_Alive() {
	c.clAssertState("controlLoop_Alive", lifecycleAlive)

	for {
		select {
		case <-c.requestStopCh:
			return
		case req := <-c.requestLaunchCh:
			c.clAliveDoLaunch(req)
		}
	}
}

func (c *Controller) clAliveDoLaunch(req launchRequest) {
	defer close(req.doneCh)

	// Up in the controlLoop_Alive select, we're doing a two-case channel read.
	//
	// Per the language spec, when two communication cases can proceed at the same time, the select implementation
	// will pick one case at random. This means we may be called when requestStopCh is already closed.
	//
	// As such, we do an additional pre-flight check here to ensure that we don't go off launching something when
	// we're supposed to be dying.
	select {
	case <-c.requestStopCh:
		return
	default:
	}

	// Even if Start() returned an error, it's possible that ImplRun has been started up. Accordingly, when we
	// do our shutdown process, we want to shutdown this component as well.
	c.stateMu.Lock()
	c.components = append(c.components, ownedComponent{req.name, req.comp})
	c.stateMu.Unlock()

	if err := req.comp.Start(c.ctx); err != nil {
		c.recordComponentError(req.name, "startup", err)
		c.RequestStop(nil)
		return
	}

	if err := req.comp.WaitReady(c.ctx, c.requestStopCh); err != nil {
		c.recordComponentError(req.name, "wait-ready", err)
		c.RequestStop(nil)
	}
}
