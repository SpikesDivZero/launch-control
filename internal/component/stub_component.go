package component

import (
	"context"
	"log/slog"
)

type StubComponent struct {
	ImplRegister func(context.Context, *slog.Logger, ControllerCallbacks)
	ImplStart    func(context.Context)
	ImplStop     func()
}

func (sc *StubComponent) Register(ctx context.Context, log *slog.Logger, callbacks ControllerCallbacks) {
	if sc.ImplRegister == nil {
		panic("StubComponent: Register was called, but not defined")
	}
	sc.ImplRegister(ctx, log, callbacks)
}

func (sc *StubComponent) Start(startCtx context.Context) {
	if sc.ImplStart == nil {
		panic("StubComponent: Start was called, but not defined")
	}
	sc.ImplStart(startCtx)
}

func (sc *StubComponent) Stop() {
	if sc.ImplStop == nil {
		panic("StubComponent: Stop was called, but not defined")
	}
	sc.ImplStop()
}
