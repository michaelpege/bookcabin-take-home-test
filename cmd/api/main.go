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

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
	httphandler "github.com/michaelpege/bookcabin-take-home-test/internal/handler/http"
	"github.com/michaelpege/bookcabin-take-home-test/internal/platform/config"
	"github.com/michaelpege/bookcabin-take-home-test/internal/platform/logger"
	"github.com/michaelpege/bookcabin-take-home-test/internal/provider"
	"github.com/michaelpege/bookcabin-take-home-test/internal/provider/batik"
	"github.com/michaelpege/bookcabin-take-home-test/internal/provider/garuda"
	"github.com/michaelpege/bookcabin-take-home-test/internal/provider/lion"
	"github.com/michaelpege/bookcabin-take-home-test/internal/usecase"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)

	policies := provider.DefaultPolicies()
	garudaClient := garuda.NewClient()
	lionClient := lion.NewClient()
	batikClient := batik.NewClient()
	providers := []domain.FlightProvider{
		provider.NewResilientProvider(garudaClient, policies[garudaClient.Name()]),
		provider.NewResilientProvider(lionClient, policies[lionClient.Name()]),
		provider.NewResilientProvider(batikClient, policies[batikClient.Name()]),
	}
	flightAggregator := usecase.NewFlightAggregator(providers, log)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      httphandler.NewRouter(log, flightAggregator),
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
