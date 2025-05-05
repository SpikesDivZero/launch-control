package ssw

import (
	"context"
	"time"
)

type StartStopWrapper struct {
	ImplStart func(context.Context) error
	ImplStop  func(context.Context) error

	StartTimeout time.Duration
	StopTimeout  time.Duration
}

func (ssw *StartStopWrapper) Run(ctx context.Context) error      { panic("NYI") }
func (ssw *StartStopWrapper) Shutdown(ctx context.Context) error { panic("NYI") }
