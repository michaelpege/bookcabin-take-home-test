package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	_ "time/tzdata" // Embed IANA timezone database for minimal container runtimes.

	httphandler "github.com/michaelpege/bookcabin-take-home-test/internal/handler/http"
	"github.com/michaelpege/bookcabin-take-home-test/internal/platform/config"
	"github.com/michaelpege/bookcabin-take-home-test/internal/platform/logger"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      httphandler.NewRouter(log),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("server_starting", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Info("shutdown_signal_received", slog.String("signal", sig.String()))
	case err := <-errCh:
		if err != nil {
			log.Error("server_failed", slog.Any("error", err))
			os.Exit(1)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("graceful_shutdown_failed", slog.Any("error", err))
		os.Exit(1)
	}
	log.Info("server_stopped")
}
