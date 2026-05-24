package controller

//go:generate go tool stringer -type=runState -trimprefix runState
type runState int

const (
	// A controller starts in the New state. Nothing has been launched.
	//
	// Transitions to:
	// - Alive: Upon the first Launch request coming in.
	// - Dead: If RequestStop is called before any Launch request comes in.
	runStateNew runState = iota

	// A controller in the Alive state indicates that one or more components have been launched, and the controller
	// is still accepting Launch requests.
	//
	// Transitions to:
	// - Dying: Upon RequestStop being called for the first time.
	runStateAlive

	// A controller in the Dying state indicates that the controller is running, however it is no longer accepting
	// Launch requests. Instead, during this state, we're shutting down all running components.
	//
	// Transitions to:
	// - Dead: Once all Components have finished shutting down.
	runStateDying

	// A controller's terminal state is Dead. In this state, the controller has finished shutting down all components.
	// It's no longer accepting Launch requests.
	//
	// Aside: If any components are still running, it's because they busted through a timeout.
	runStateDead
)
