package controller

import (
	"fmt"
	"time"
)

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

const dyingMonitorExitReportingGracePeriod = 100 * time.Millisecond

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

	// In some cases, and especially when exercised by the race detector, the doneCh is closed before all the
	// monitorExit coroutines had a chance to report their final status back to the controller.
	//
	// This resulted in [Controller.Wait] erroneously returning "no error" to the caller, even if a user-provided
	// [ImplRun] actually returned an error.
	//
	// By adding a short delay before closing the doneCh (via defer), we effectively yield to allow the go scheduler
	// to run all waiting coroutines.
	//
	// Why not [runtime.Gosched]? I may reconsider it's use it in the future, but for now, it doesn't provide
	// sufficient assurance that all waiting coroutines will execute (esp. when either the race detector is enabled,
	// or when GOMAXPROCS > 1 and an internal reschedule/rebalance happens between go processors)
	//
	// No, a sleep isn't perfect, but it's arguably better than a Gosched call.
	time.Sleep(dyingMonitorExitReportingGracePeriod)
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
