package component

import (
	"context"
	"errors"
)

func (c *Component) waitReady(ctx context.Context) error {
	if c.ImplCheckReady == nil {
		return nil
	}

	// FIXME: Placeholder so we can test startup
	ready, err := c.ImplCheckReady(ctx)
	if ready && err == nil {
		return nil
	}
	if c.CheckReadyOptions.MaxAttempts == 1 {
		return errors.New("failed to become ready")
	}
	panic("NYI: waitReady only has minimal placeholder code currently")
}
