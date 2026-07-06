// Command api is the composition root: the only place that knows concrete
// adapters and wires them to the use cases before starting the HTTP server.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpapi "github.com/orhan17/booking-aggregator/internal/adapters/inbound/http"
	"github.com/orhan17/booking-aggregator/internal/adapters/outbound/operators"
	"github.com/orhan17/booking-aggregator/internal/adapters/outbound/persistence"
	"github.com/orhan17/booking-aggregator/internal/app"
	"github.com/orhan17/booking-aggregator/internal/ports"
)

const (
	defaultAddr   = ":8080"
	searchTimeout = 3 * time.Second
	shutdownGrace = 10 * time.Second
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ops := []ports.OperatorPort{
		operators.NewAnex(),
		operators.NewCoral(),
		operators.NewSunmar(),
	}
	repo := persistence.NewMemoryBookingRepository()

	searchSvc := app.NewSearchService(ops, logger, searchTimeout, len(ops), time.Now)
	bookingSvc := app.NewBookingService(ops, repo, logger, time.Now, newBookingID)

	router := httpapi.NewRouter(searchSvc, bookingSvc, logger)

	srv := &http.Server{
		Addr:              envOr("ADDR", defaultAddr),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if err := run(srv, logger); err != nil {
		logger.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

func run(srv *http.Server, logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		logger.Info("shutdown signal received, draining connections")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func newBookingID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "bk_" + hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
	}
	return "bk_" + hex.EncodeToString(b[:])
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
