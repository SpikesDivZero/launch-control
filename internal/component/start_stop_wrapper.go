package component

import (
	"context"
	"sync"
	"time"
)

type StartStopWrapper struct {
	ImplStart func(context.Context) error
	ImplStop  func(context.Context) error

	StartTimeout time.Duration
	StopTimeout  time.Duration

	stateMu       sync.Mutex
	requestStopCh chan struct{}
}

func (ssw *StartStopWrapper) Run(ctx context.Context) error {
	ssw.initForRun()

	if err := ssw.doCall(ctx, "StartStopWrapper.StartTimeout", ssw.StartTimeout, ssw.ImplStart); err != nil {
		return err
	}

	<-ssw.requestStopCh

	return ssw.doCall(ctx, "StartStopWrapper.StopTimeout", ssw.StopTimeout, ssw.ImplStop)
}

func (ssw *StartStopWrapper) initForRun() {
	ssw.stateMu.Lock()
	defer ssw.stateMu.Unlock()

	if ssw.requestStopCh != nil {
		panic("internal: StartStopWrapper Run called twice")
	}
	ssw.requestStopCh = make(chan struct{})
}

func (ssw *StartStopWrapper) Shutdown(ctx context.Context) error {
	ssw.stateMu.Lock()
	defer ssw.stateMu.Unlock()

	if ssw.requestStopCh == nil {
		ssw.requestStopCh = make(chan struct{})
	}
	close(ssw.requestStopCh)
	return nil
}

func (ssw *StartStopWrapper) doCall(
	ctx context.Context,
	timeoutSource string,
	timeout time.Duration,
	impl func(context.Context) error,
) error {
	err, callErr := (<-AsyncCall(ctx, timeoutSource, timeout, 100*time.Millisecond, impl)).Values()
	if callErr != nil {
		return callErr
	}
	return err
}
