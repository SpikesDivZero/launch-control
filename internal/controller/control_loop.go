package controller

func init() { _ = (&Controller{}).controlLoop } // FIXME: Placeholder to silence warnings while sketching

//go:generate go tool stringer -type lifecycleState -trimprefix lifecycle
type lifecycleState int

const (
	lifecycleNew lifecycleState = iota
	lifecycleAlive
	lifecycleDying
	lifecycleDead
)

// The main entry point for our controlLoop. It's job is just to call the different lifecycle stages in order.
// If we skip from New->Dying, then this shouldn't ever be invoked?
func (c *Controller) controlLoop() {
	_ = lifecycleAlive.String()
	_ = c.controlLoop_Alive
	_ = c.controlLoop_Dying
	panic("NYI")
}
