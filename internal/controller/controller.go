package controller

import (
	"context"
	"log/slog"
)

type Component interface {
	ConnectController(
		log *slog.Logger,
		notifyOnExited func(error),
	)
	Start(context.Context) error
	Shutdown(context.Context) error
}

type Controller struct{}

func New(context.Context, *slog.Logger) *Controller {
	return &Controller{}
}

func (c *Controller) Launch(name string, comp Component) { panic("NYI") }
func (c *Controller) RequestStop(reason error)           { panic("NYI") }
func (c *Controller) Wait() error                        { panic("NYI") }
func (c *Controller) Err() error                         { panic("NYI") }
