package e2etest

import (
	"context"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	launchcontrol "github.com/spikesdivzero/launch-control"
)

type eetState struct {
	ctx       context.Context
	cancelCtx context.CancelCauseFunc

	log *slog.Logger
}

// Runs an end-to-end test inside of a synctest bubble.
//
// A Controller is created for you, and the test state contains some amenities to make life easier.
//
// After the test runs, it waits 10 simulated seconds, so that the controller's hidden cleanup has time to wrap up.
// (Without this pause, synctest will panic for a leaked goroutine.)
func eeTest(t *testing.T, f func(*testing.T, *launchcontrol.Controller, eetState)) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(t.Context())
		defer cancel(nil)

		log := slog.New(slog.NewJSONHandler(t.Output(), &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		ctrl := launchcontrol.New(ctx)
		ctrl.SetLogger(log)

		f(t, ctrl, eetState{
			ctx:       ctx,
			cancelCtx: cancel,

			log: log,
		})

		time.Sleep(10 * time.Second)
	})
}
