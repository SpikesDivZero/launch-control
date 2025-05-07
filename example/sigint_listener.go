package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
)

// Even if you're using an HTTP-based hook to request a shutdown of your application, you may
// still want a SIGINT handler, so that your devs can verify shutdown beaviors locally (without
// needing to run a separate curl request).
type IntteruptListener struct {
	Log *slog.Logger

	runCtxCancel context.CancelFunc
}

func (il *IntteruptListener) Run(ctx context.Context) error {
	ctx, il.runCtxCancel = context.WithCancel(ctx)
	defer il.runCtxCancel()

	signalCh := make(chan os.Signal, 1)
	defer close(signalCh)

	signal.Notify(signalCh, os.Interrupt)
	defer signal.Stop(signalCh)

	select {
	case <-signalCh:
		il.Log.Info("Received SIGINT")
	case <-ctx.Done():
		il.Log.Info("Context cancelled (could be via Shutdown)")
	}

	// This demonstrates one way of shutting down the stack.
	//
	// The controller considers all components under it's perview to be necessary for the
	// application to operate successfully.
	//
	// If the controller is alive (not Terminating) and a blocking Run() method returns,
	// then the controller considers the application to be failing, and initiates a shutdown
	// process.
	//
	// This allows for this instance of the application to exit, and the in-use process management
	// solution (systemd, k8s, whatever) can then relaunch the application to start again.
	return nil
}

func (il *IntteruptListener) Shutdown(ctx context.Context) error {
	if il.runCtxCancel == nil {
		return errors.New("runCtxCancel isn't set (run not started?)")
	}

	il.runCtxCancel()
	return nil
}
