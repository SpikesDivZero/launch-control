package controller

//go:generate go tool stringer -type lifecycleState -trimprefix lifecycle
type lifecycleState int

const (
	lifecycleNew lifecycleState = iota
	lifecycleAlive
	lifecycleDying
	lifecycleDead
)

type launchRequest struct {
	name   string
	comp   Component
	doneCh chan struct{}
}

// The main entry point for our controlLoop. It's job is just to call the different lifecycle stages in order.
// If we skip from New->Dying, then this shouldn't ever be invoked?
func (c *Controller) controlLoop() {
	_ = c.controlLoop_Alive
	_ = c.controlLoop_Dying

	// Dummy placeholder code
	defer close(c.doneCh)
	for {
		select {
		case <-c.requestStopCh:
			return
		case req := <-c.requestLaunchCh:
			close(req.doneCh)
		}
	}
}
