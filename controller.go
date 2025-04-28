package launch

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/spikesdivzero/launch-control/internal/controller"
)

// The bulk of the controller is implemented internally.
// We expose a documented struct here, as opposed to an interface, primarily for easier reading and godoc's sake.

type Controller struct {
	impl *controller.Controller
}

func NewController(ctx context.Context, log *slog.Logger) Controller {
	return Controller{impl: controller.New(ctx, log)}
}

func (c *Controller) Launch(name string, opts ComponentOption) {
	comp, err := buildComponent(name, opts)
	if err != nil {
		panic(fmt.Sprintf("component build failed: %v", err))
	}
	c.impl.Launch(comp)
}

func (c *Controller) RequestStop(reason error) {
	c.impl.RequestStop(reason)
}

func (c *Controller) Wait() error {
	return c.impl.Wait()
}

func (c *Controller) Err() error {
	return c.impl.Err()
}
