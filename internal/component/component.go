package component

import (
	"context"
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
	logError       func(stage string, err error)
	notifyOnExited func(error)

	// Lifecycle-related state, created in [Start]
	runCtxCancel context.CancelFunc
	doneCh       <-chan struct{}
}

func (c *Component) ConnectController(
	logError func(stage string, err error),
	notifyOnExited func(error),
) {
	c.logError = logError
	c.notifyOnExited = notifyOnExited
}
