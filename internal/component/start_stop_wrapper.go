package component

import "context"

type StartStopWrapper struct {
	ImplStart func(context.Context) error
	ImplStop  func(context.Context) error

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

	panic("NYI: SSW.Run")
}

func (ssw *StartStopWrapper) Stop(ctx context.Context) error {
	if fn := ssw.TestControl.MockStop; fn != nil {
		return fn(ctx)
	}

	panic("NYI: SSW.Stop")
}
