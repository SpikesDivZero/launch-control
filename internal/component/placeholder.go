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
}

type StartStopWrapper struct {
	ImplStart func(context.Context) error
	ImplStop  func(context.Context) error

	StartTimeout time.Duration
	StopTimeout  time.Duration
}
