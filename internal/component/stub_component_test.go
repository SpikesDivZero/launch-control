package component_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/shoenig/test"
	"github.com/spikesdivzero/launch-control/internal/component"
	"github.com/spikesdivzero/launch-control/internal/testutil"
)

func TestStubComponent_Register(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		defer testutil.WantPanic(t, "StubComponent: Register was called, but not defined")
		sc := component.StubComponent{}
		sc.Register(t.Context(), nil, component.ControllerCallbacks{})
	})

	t.Run("happy", func(t *testing.T) {
		registerCalled := false
		sc := component.StubComponent{
			ImplRegister: func(ctx context.Context, l *slog.Logger, cc component.ControllerCallbacks) {
				registerCalled = true
			},
		}
		sc.Register(t.Context(), nil, component.ControllerCallbacks{})
		test.True(t, registerCalled)
	})
}

func TestStubComponent_Start(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		defer testutil.WantPanic(t, "StubComponent: Start was called, but not defined")
		sc := component.StubComponent{}
		sc.Start(t.Context())
	})

	t.Run("happy", func(t *testing.T) {
		startCalled := false
		sc := component.StubComponent{
			ImplStart: func(ctx context.Context) { startCalled = true },
		}
		sc.Start(t.Context())
		test.True(t, startCalled)
	})
}

func TestStubComponent_Stop(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		defer testutil.WantPanic(t, "StubComponent: Stop was called, but not defined")
		sc := component.StubComponent{}
		sc.Stop()
	})

	t.Run("happy", func(t *testing.T) {
		stopCalled := false
		sc := component.StubComponent{
			ImplStop: func() { stopCalled = true },
		}
		sc.Stop()
		test.True(t, stopCalled)
	})
}
