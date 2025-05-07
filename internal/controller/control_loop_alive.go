package controller

// The contents of this file run when lifecycleState is lifecycleAlive.
//
// I've split it into different files based on stages both for consistency with the component code,
// as well as clarity in the event that I need to extend this. (For instance, I may in the future
// add in the ability to launch a group of components concurrently.)

func (c *Controller) controlLoop_Alive() {
	// Dummy placeholder code
	for {
		select {
		case <-c.requestStopCh:
			return
		case req := <-c.requestLaunchCh:
			close(req.doneCh)
		}
	}
}
