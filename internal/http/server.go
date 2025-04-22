package http

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Middleware func(http.Handler) http.Handler

type ApiServer struct {
	server     *http.Server
	mux        *http.ServeMux
	middleware []Middleware
	logger     *slog.Logger
}

func NewApiServer(port int, logger *slog.Logger) *ApiServer {
	mux := http.NewServeMux()
	return &ApiServer{
		mux:        mux,
		middleware: []Middleware{},
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: h2c.NewHandler(mux, &http2.Server{}),
		},
		logger: logger,
	}
}

func (a *ApiServer) Use(mw Middleware) {
	a.middleware = append(a.middleware, mw)
}

func (a *ApiServer) applyMiddleware(h http.Handler) http.Handler {
	for i := len(a.middleware) - 1; i >= 0; i-- {
		h = a.middleware[i](h)
	}
	return h
}

func (a *ApiServer) AddHandler(path string, hFunc func(w http.ResponseWriter, r *http.Request)) {
	finalHandler := a.applyMiddleware(http.HandlerFunc(hFunc))
	a.mux.Handle(path, finalHandler)
}

func (a *ApiServer) AddStaticHandler(urlPath string, dirPath string) {
	// FileServer to serve static files
	fileServer := http.FileServer(http.Dir(dirPath))

	// Custom handler to serve .wasm files with correct Content-Type
	a.mux.Handle(urlPath, http.StripPrefix(urlPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for all static file requests
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		fileServer.ServeHTTP(w, r)
	})))
}

func (a *ApiServer) AddFileHandler(path string, filePath string, mimeType string) {
	a.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		// make sure it’s always the same YAML MIME‑type
		w.Header().Set("Content-Type", mimeType)
		http.ServeFile(w, r, filePath)
	})
}

func (a *ApiServer) ListenAndServe() {

	// Channel to listen for interrupt signals (e.g., Ctrl+C)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Run the server in a goroutine
	go func() {
		a.logger.Log(context.Background(), slog.LevelInfo, "Server is running", "addr", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.logger.Log(context.Background(), slog.LevelError, "Could not listen", "addr", a.server.Addr)
			os.Exit(1)
		}
	}()

	// Block until we receive a signal
	<-stop
	a.logger.Log(context.Background(), slog.LevelInfo, "Shutting down server...")

	// Create a deadline to wait for the server to shut down gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	// Attempt to gracefully shut down the server
	if err := a.server.Shutdown(ctx); err != nil {
		a.logger.Log(ctx, slog.LevelError, "server forced to shutdown")
		os.Exit(1)
	}

	a.logger.Log(ctx, slog.LevelInfo, "Server gracefully stopped")
}
