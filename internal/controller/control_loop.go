package controller

import "fmt"

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
	defer close(c.doneCh)

	// We enter this function in the Alive state (set by sendLaunchRequest)
	c.clAssertState("controlLoop", lifecycleAlive) // trust but verify
	c.controlLoop_Alive()

	c.clSetState(lifecycleAlive, lifecycleDying)
	c.controlLoop_Dying()

	c.clSetState(lifecycleDying, lifecycleDead)
}

func (c *Controller) clAssertState(in string, want lifecycleState) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	if c.lifecycleState != want {
		panic(fmt.Sprintf("internal: %v clAssertState, state %v, expected %v", in, c.lifecycleState, want))
	}
}

func (c *Controller) clSetState(from, to lifecycleState) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	if c.lifecycleState != from {
		panic(fmt.Sprintf("internal: SetState from state %v, expected %v", c.lifecycleState, from))
	}
	c.lifecycleState = to
}
