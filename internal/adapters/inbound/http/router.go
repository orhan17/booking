// Package http is the inbound REST adapter. Its handlers translate HTTP to and
// from the inbound ports and contain no business logic.
package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/orhan17/booking-aggregator/internal/app"
	"github.com/orhan17/booking-aggregator/internal/domain"
	"github.com/orhan17/booking-aggregator/internal/ports"
)

const maxBodyBytes = 1 << 20

type handlers struct {
	search  ports.SearchUseCase
	booking ports.BookingUseCase
	logger  *slog.Logger
}

// NewRouter builds the HTTP handler, routing requests to the use cases. It
// depends only on the inbound ports, so main decides the concrete services.
func NewRouter(search ports.SearchUseCase, booking ports.BookingUseCase, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	h := &handlers{search: search, booking: booking, logger: logger}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.handleHealth)
	mux.HandleFunc("POST /search", h.handleSearch)
	mux.HandleFunc("POST /bookings", h.handleCreateBooking)
	mux.HandleFunc("GET /bookings/{id}", h.handleGetBooking)
	return mux
}

func (h *handlers) handleHealth(w http.ResponseWriter, _ *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func (h *handlers) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.logger.Error("encode response failed", "error", err)
	}
}

func (h *handlers) writeError(w http.ResponseWriter, status int, err error) {
	msg := err.Error()
	if status >= http.StatusInternalServerError {
		h.logger.Error("request failed", "status", status, "error", err)
		msg = "internal server error"
	}
	h.writeJSON(w, status, errorResponse{Error: msg})
}

// statusFor maps a use-case error to an HTTP status, interpreting the ports'
// error contract.
func statusFor(err error) int {
	switch {
	case errors.Is(err, domain.ErrBookingNotFound):
		return http.StatusNotFound
	case errors.Is(err, app.ErrUnknownOperator):
		return http.StatusUnprocessableEntity
	case isValidationError(err):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

var validationErrors = []error{
	domain.ErrEmptyOrigin,
	domain.ErrEmptyDestination,
	domain.ErrSameOriginDestination,
	domain.ErrMissingDepartureDate,
	domain.ErrDepartureInPast,
	domain.ErrReturnBeforeDeparture,
	domain.ErrNoPassengers,
	domain.ErrTooManyPassengers,
	domain.ErrEmptyBookingID,
}

func isValidationError(err error) bool {
	for _, target := range validationErrors {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
