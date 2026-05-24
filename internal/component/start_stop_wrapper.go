package component

import (
	"context"
	"errors"
)

type StartStopWrapper struct {
	Comp      *Component
	ImplStart func(context.Context) error
	ImplStop  func(context.Context) error

	requestStopCh chan struct{}

	// I don't want to do a full mock for a couple tests, so this'll have to suffice.
	TestControl struct {
		MockRun  func(context.Context) error
		MockStop func(context.Context) error
	}
}

func (ssw *StartStopWrapper) Run(ctx context.Context) error {
	if fn := ssw.TestControl.MockRun; fn != nil {
		return fn(ctx)
	}

	if ssw.requestStopCh != nil {
		panic("SSW.Run called more than once?")
	}
	ssw.requestStopCh = make(chan struct{})

	if err := ssw.ImplStart(ctx); err != nil {
		return WrapComponentError(ssw.Comp, "Start", err)
	}

	<-ssw.requestStopCh

	return WrapComponentError(ssw.Comp, "Stop", ssw.ImplStop(ctx))
}

func (ssw *StartStopWrapper) Stop(ctx context.Context) error {
	if fn := ssw.TestControl.MockStop; fn != nil {
		return fn(ctx)
	}

	if ssw.requestStopCh == nil {
		// TODO: Revisit. This may be better off as a "log" instead of an error.
		// I'm not sure how/when this would reasonably happen, but I'd like to know if it does happen.
		return WrapComponentError(ssw.Comp, "Stop",
			errors.New("StartStopWrapper: Stop called, but Start was never invoked"))
	}

	select {
	case <-ssw.requestStopCh:
		panic("SSW.Stop called more than once?")
	default:
		close(ssw.requestStopCh)
	}

	return nil
}
