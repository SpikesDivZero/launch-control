package controller

import (
	"context"
	"log/slog"

	"github.com/spikesdivzero/launch-control/internal/component"
)

type Controller struct{}

func New(context.Context, *slog.Logger) *Controller {
	return &Controller{}
}

func (c *Controller) Launch(comp *component.Component) { panic("NYI") }
func (c *Controller) RequestStop(reason error)         { panic("NYI") }
func (c *Controller) Wait() error                      { panic("NYI") }
func (c *Controller) Err() error                       { panic("NYI") }
