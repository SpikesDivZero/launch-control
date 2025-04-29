package component

import "context"

func (c *Component) Start(ctx context.Context) error {
	if c.doneCh != nil {
		panic("Start called twice?")
	}
	c.doneCh = make(chan struct{})
	ctx, c.runCtxCancel = context.WithCancel(ctx)

	_ = ctx
	panic("NYI")
}
