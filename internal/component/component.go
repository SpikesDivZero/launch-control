package component

import (
	"context"
	"log/slog"
	"time"
)

type ShutdownOptions struct {
	CallTimeout       time.Duration
	CompletionTimeout time.Duration
}

type CheckReadyOptions struct {
	CallTimeout time.Duration
	Backoff     func() time.Duration
	MaxAttempts int
}

type Component struct {
	Name string

	ImplRun func(context.Context) error

	ImplShutdown    func(context.Context) error
	ShutdownOptions ShutdownOptions

	ImplCheckReady    func(context.Context) (bool, error)
	CheckReadyOptions CheckReadyOptions

	// Values provided by by [ConnectController]
	log            *slog.Logger
	notifyOnExited func(*Component, error)

	// Lifecycle-related state, created in [Start]
	runCtxCancel context.CancelFunc
	doneCh       <-chan struct{}
}

func (c *Component) ConnectController(
	log *slog.Logger,
	notifyOnExited func(*Component, error),
) {
	c.log = log
	c.notifyOnExited = notifyOnExited
}
