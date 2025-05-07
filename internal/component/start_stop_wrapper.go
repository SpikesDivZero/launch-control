package component

import (
	"context"
	"time"
)

type StartStopWrapper struct {
	ImplStart func(context.Context) error
	ImplStop  func(context.Context) error

	StartTimeout time.Duration
	StopTimeout  time.Duration

	requestStopCh chan struct{}
}

func (ssw *StartStopWrapper) Run(ctx context.Context) error {
	if ssw.requestStopCh != nil {
		panic("internal: StartStopWrapper Run called twice")
	}
	ssw.requestStopCh = make(chan struct{})

	if err := ssw.doCall(ctx, ssw.StartTimeout, ssw.ImplStart); err != nil {
		return err
	}

	<-ssw.requestStopCh

	return ssw.doCall(ctx, ssw.StopTimeout, ssw.ImplStop)
}

func (ssw *StartStopWrapper) Shutdown(ctx context.Context) error {
	if ssw.requestStopCh == nil {
		ssw.requestStopCh = make(chan struct{})
	}
	close(ssw.requestStopCh)
	return nil
}

func (ssw *StartStopWrapper) doCall(
	ctx context.Context,
	timeout time.Duration,
	impl func(context.Context) error,
) error {
	err, callErr := (<-AsyncCall(ctx, timeout, 100*time.Millisecond, impl)).Values()
	if callErr != nil {
		return callErr
	}
	return err
}
