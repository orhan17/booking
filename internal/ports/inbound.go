package ports

import (
	"context"

	"github.com/orhan17/booking-aggregator/internal/domain"
)

// SearchUseCase is the inbound port for aggregating operator offers.
type SearchUseCase interface {
	Search(ctx context.Context, criteria domain.SearchCriteria) ([]domain.Offer, error)
}

// BookRequest is the input to BookingUseCase.Book: the selected offer and the
// number of passengers to book it for.
type BookRequest struct {
	Offer      domain.Offer
	Passengers int
}

// BookingUseCase is the inbound port for booking an offer and reading a booking.
type BookingUseCase interface {
	Book(ctx context.Context, req BookRequest) (*domain.Booking, error)
	Get(ctx context.Context, id string) (*domain.Booking, error)
}
