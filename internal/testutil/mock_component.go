package testutil

import (
	"context"
	"time"
)

// For testing the controller, I'd like a trivial mock for the controller.Component interface.
// There's an argument to be made in favor of full mocking frameworks, but I kinda like the
// practice and simplicity of a thing that can simulate delays.

type MockComponent struct {
	StartOptions struct {
		Hook  func()
		Sleep time.Duration
		Err   error
	}

	ShutdownOptions struct {
		Hook  func()
		Sleep time.Duration
		Err   error
	}

	WaitReadyOptions struct {
		Hook  func()
		Sleep time.Duration
		Err   error
	}

	Recorder struct {
		Connect struct {
			Called           bool
			LogError         func(string, error)
			NotifyOnExited   func(error)
			AsyncGracePeriod time.Duration
		}
		Start struct {
			Called bool
			Ctx    context.Context
		}
		Shutdown struct {
			Called bool
			Ctx    context.Context
		}
		WaitReady struct {
			Called      bool
			Ctx         context.Context
			AbortLoopCh <-chan struct{}
		}
	}
}

func (mc *MockComponent) ConnectController(
	logError func(string, error),
	notifyOnExited func(error),
	asyncGracePeriod time.Duration,
) {
	rc := &mc.Recorder.Connect
	rc.Called = true
	rc.LogError = logError
	rc.NotifyOnExited = notifyOnExited
	rc.AsyncGracePeriod = asyncGracePeriod
}

func (mc *MockComponent) Start(ctx context.Context) error {
	rc := &mc.Recorder.Start
	rc.Called = true
	rc.Ctx = ctx

	if hook := mc.StartOptions.Hook; hook != nil {
		hook()
	}
	time.Sleep(mc.StartOptions.Sleep)
	return mc.StartOptions.Err
}

func (mc *MockComponent) Shutdown(ctx context.Context) error {
	rc := &mc.Recorder.Shutdown
	rc.Called = true
	rc.Ctx = ctx

	if hook := mc.ShutdownOptions.Hook; hook != nil {
		hook()
	}
	time.Sleep(mc.ShutdownOptions.Sleep)
	return mc.ShutdownOptions.Err
}

func (mc *MockComponent) WaitReady(ctx context.Context, abortLoopCh <-chan struct{}) error {
	rc := &mc.Recorder.WaitReady
	rc.Called = true
	rc.Ctx = ctx
	rc.AbortLoopCh = abortLoopCh

	if hook := mc.WaitReadyOptions.Hook; hook != nil {
		hook()
	}
	time.Sleep(mc.WaitReadyOptions.Sleep)
	return mc.WaitReadyOptions.Err
}
