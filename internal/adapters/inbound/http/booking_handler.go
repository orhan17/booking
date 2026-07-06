package http

import (
	"net/http"

	"github.com/orhan17/booking-aggregator/internal/ports"
)

// handleCreateBooking books an offer. A booking that resolves to failed (e.g.
// price changed) is still a created resource, so both confirmed and failed
// bookings return 201; only transport-level problems are error statuses.
func (h *handlers) handleCreateBooking(w http.ResponseWriter, r *http.Request) {
	var req bookingRequest
	if err := decodeJSON(w, r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, err)
		return
	}

	booking, err := h.booking.Book(r.Context(), ports.BookRequest{
		Offer:      req.Offer.toDomain(),
		Passengers: req.Passengers,
	})
	if err != nil {
		h.writeError(w, statusFor(err), err)
		return
	}

	w.Header().Set("Location", "/bookings/"+booking.ID())
	h.writeJSON(w, http.StatusCreated, bookingFromDomain(booking))
}

func (h *handlers) handleGetBooking(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	booking, err := h.booking.Get(r.Context(), id)
	if err != nil {
		h.writeError(w, statusFor(err), err)
		return
	}

	h.writeJSON(w, http.StatusOK, bookingFromDomain(booking))
}
