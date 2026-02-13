package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/dpotapov/slogpfx"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-colorable"

	"github.com/spikesdivzero/launch-control"
)

// This isn't really necessary for the example, but it helps a bit to be able to see some color
// when building/debugging everything.
func colorLogger() *slog.Logger {
	h := tint.NewHandler(colorable.NewColorable(os.Stdout), &tint.Options{
		Level: slog.LevelDebug,
	})

	prefixed := slogpfx.NewHandler(h, &slogpfx.HandlerOptions{
		PrefixKeys: []string{"prefix"},
	})

	return slog.New(prefixed)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := colorLogger()

	// You can combine any number of options here to create a default option set.
	// Caveat: run styles and check-ready should not be here, as they can only be provided once per component.
	defaultOpts := launch.Options126{}

	ctrl := launch.NewController(ctx)
	ctrl.SetLogger(log)

	sigint := IntteruptListener{
		Log: log.With("prefix", "IntteruptListener"),
	}
	ctrl.Launch126("sigint",
		defaultOpts,
		launch.Options126{
			Run:      sigint.Run,
			Shutdown: sigint.Shutdown,
		},
	)

	mgmt := NewHttpMgmtServer(
		log.With("prefix", "http:mgmt"),
		func() { ctrl.RequestStop(errors.New("stop requested via http mgmt")) },
	)
	ctrl.Launch126("http-mgmt",
		defaultOpts,
		launch.Options126{
			Run:      mgmt.Run,
			Shutdown: mgmt.Shutdown,
		},
	)

	data := DataConnector{Log: log.With("prefix", "datastore")}
	ctrl.Launch126("data",
		defaultOpts,
		launch.Options126{
			Start:      data.Connect,
			Stop:       data.Disconnect,
			CheckReady: data.CheckReady,
		},
	)

	app := NewHttpAppServer(log.With("prefix", "http:app"))
	ctrl.Launch126("http-app",
		defaultOpts,
		launch.Options126{
			Run:      app.Run,
			Shutdown: app.Shutdown,
		},
	)

	// This one's a bit of an odd one, but it exists to show how we
	ctrl.Launch126("ready-state",
		defaultOpts,
		launch.Options126{
			Start: func(ctx context.Context) error {
				// Once we start up, we're ready to accept traffic
				mgmt.setReadyState(true)
				return nil
			},
			Stop: func(ctx context.Context) error {
				// And as we're shutting down, we're no longer willing to accept traffic.
				mgmt.setReadyState(false)

				// After marking the service as no longer ready to receive traffic, we should
				// wait before proceeding to shut down the rest of the way.
				//
				// How long exactly is a question I won't presume to answer for you.
				time.Sleep(5 * time.Second)

				return nil
			},
		},
	)

	log.Info("Started up; you can cancel it via ^C or curl http://localhost:8844/_/shutdown")
	if err := ctrl.Wait(); err != nil {
		log.Error("Controller wait returned an error", "err", err)
	}
}
