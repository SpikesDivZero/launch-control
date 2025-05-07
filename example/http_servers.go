package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type httpServer struct {
	Log *slog.Logger

	addr string
	mux  *http.ServeMux

	srv *http.Server
}

func newBaseHttpServer(log *slog.Logger, addr string) *httpServer {
	return &httpServer{
		Log: log,

		addr: addr,
		mux:  http.NewServeMux(),
	}
}

func (h *httpServer) Run(ctx context.Context) error {
	h.srv = &http.Server{
		Addr:    h.addr,
		Handler: h.mux,
	}

	err := h.srv.ListenAndServe()
	if err == nil || err == http.ErrServerClosed {
		return nil
	}
	return fmt.Errorf("http.Server.ListenAndServe returned %w", err)
}

func (h *httpServer) Shutdown(ctx context.Context) error {
	err := h.srv.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("http.Server.Shutdown returned %w", err)
	}

	return nil
}

type httpMgmtServer struct {
	httpServer

	appReady bool
}

func NewHttpMgmtServer(log *slog.Logger, requestShutdown func()) *httpMgmtServer {
	s := &httpMgmtServer{
		httpServer: *newBaseHttpServer(log, ":8844"),
	}

	s.mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("example mgmt server"))
	}))

	// An alternate way to shutdown the service, for example via a k8s hook.
	//
	// This probably should be limited to POST, but for demo purposes, all methods work.
	s.mux.Handle("/_/shutdown", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestShutdown()
		w.Write([]byte("shutdown requested"))
	}))

	s.mux.Handle("/_/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// As long as the app is alive, then it's healty. A real app may have different
		// conditions.
		w.Write([]byte("happy as a clam"))
	}))

	// Your application can be healthy without being ready/willing to accept new traffic.
	//
	// Let's assume that you're using some sort of layer 7 traffic management software that
	// supports endpoint readiness probes (envoy, haproxy, whatever).
	//
	// In this case, you wouldn't want to advertise your service as ready to accept traffic
	// until e.g. you've loaded any runtime/dynamic config, and successfully verified your
	// connnection to your data stores.
	//
	// Additionally, when shutting down, you may want to mark the service as "not ready", and
	// wait a bit for the router to detect the status. Only then might you want to proceed
	// with a normal graceful shutdown, to minimize the chance of any undesirable errors.
	s.mux.Handle("/_/ready", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.appReady {
			w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("please stop sending traffic here"))
		}
	}))

	s.mux.Handle("/_/metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("your metrics output"))
	}))

	return s
}

func (s *httpMgmtServer) setReadyState(v bool) {
	s.appReady = v
}

func NewHttpAppServer(log *slog.Logger) *httpServer {
	s := newBaseHttpServer(log, ":8845")

	s.mux.HandleFunc("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("your application stuff would be here"))
	}))

	// Exists to help demonstrate a graceful shutdown
	s.mux.HandleFunc("/long-request", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Log.Info("long request started: sleeping 15s")
		time.Sleep(15 * time.Second)
		s.Log.Info("long request sleep ended")
		w.Write([]byte("that took a while"))
	}))

	return s
}
