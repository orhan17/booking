package ports

import (
	"context"

	"github.com/orhan17/booking-aggregator/internal/domain"
)

// ReconcileResult is an operator's pre-booking answer: whether the offer is
// still bookable and its current (possibly changed) price.
type ReconcileResult struct {
	Available bool
	Offer     domain.Offer
}

// OperatorPort is a driven port for one tour operator the service integrates
// with. Adding an operator means adding an adapter that satisfies it.
type OperatorPort interface {
	Name() string
	Search(ctx context.Context, criteria domain.SearchCriteria) ([]domain.Offer, error)
	Reconcile(ctx context.Context, offer domain.Offer) (ReconcileResult, error)
	Book(ctx context.Context, offer domain.Offer, passengers int) (confirmationRef string, err error)
}

// BookingRepository persists the Booking aggregate. FindByID returns
// domain.ErrBookingNotFound when no booking exists for the id.
type BookingRepository interface {
	Save(ctx context.Context, booking *domain.Booking) error
	FindByID(ctx context.Context, id string) (*domain.Booking, error)
}
