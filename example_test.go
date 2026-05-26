package launchcontrol

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func Example() {
	ctx := context.TODO()
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	ctrl := New(ctx)
	ctrl.SetLogger(log)

	// A way for devs to easily test the shutdown behaviors.
	signalCh := make(chan os.Signal, 1)
	ctrl.Launch("sigint", Options{
		Run: func(ctx context.Context) error {
			signal.Notify(signalCh, os.Interrupt)
			<-signalCh
			signal.Stop(signalCh)
			// Run returning triggers a shutdown
			return nil
		},
		Stop: func(ctx context.Context) error {
			signal.Stop(signalCh)
			return nil
		},
	})

	// A simple HTTP server; has the ability to shutdown via HTTP request, and signal if the service is ready.
	var httpSrv http.Server
	var httpReadyFlag bool
	ctrl.Launch("http", Options{
		Run: func(ctx context.Context) error {
			mux := http.NewServeMux()
			mux.HandleFunc("/_/ready", func(w http.ResponseWriter, r *http.Request) {
				if httpReadyFlag {
					w.WriteHeader(204)
				} else {
					w.WriteHeader(503)
				}
			})
			mux.HandleFunc("/_/shutdown", func(w http.ResponseWriter, r *http.Request) {
				ctrl.RequestStop(nil)
				w.WriteHeader(204)
			})
			httpSrv = http.Server{
				Addr:    ":8080",
				Handler: mux,
			}
			if err := httpSrv.ListenAndServe(); err == http.ErrServerClosed {
				return nil
			} else {
				return err
			}
		},
		Stop: func(ctx context.Context) error {
			return httpSrv.Shutdown(ctx)
		},
	})

	// And last but not least, we change the ready state as we startup and shutdown.
	ctrl.Launch("ready-state", Options{
		Start: func(ctx context.Context) error {
			httpReadyFlag = true

			fmt.Println("Application online. To quit, either:")
			fmt.Println("* press ^C in terminal (dev)")
			fmt.Println("* GET http://localhost:8080/_/shutdown")

			return nil
		},
		Stop: func(ctx context.Context) error {
			httpReadyFlag = false

			// Wait after changing the ready state so our traffic management (envoy, isito, whatever) can detect
			// that we're no longer ready and stop sending traffic our way.
			fmt.Println("Waiting 15s for reverse proxies to notice us")
			time.Sleep(15 * time.Second)

			return nil
		},
	})

	if err := ctrl.Wait(); err != nil {
		log.Error("Launch Controller returned an error", "err", slog.AnyValue(err))
	}
}
