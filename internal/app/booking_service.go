package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/orhan17/booking-aggregator/internal/domain"
	"github.com/orhan17/booking-aggregator/internal/ports"
)

// BookingService reserves a selected offer. It reconciles availability and price
// with the operator before confirming; any business-level obstacle results in a
// persisted failed booking rather than a transport error, so the caller always
// learns the outcome by id.
type BookingService struct {
	operators map[string]ports.OperatorPort
	repo      ports.BookingRepository
	logger    *slog.Logger
	now       func() time.Time
	newID     func() string
}

var _ ports.BookingUseCase = (*BookingService)(nil)

// NewBookingService wires the booking use case. Operators are indexed by Name so
// a booking routes to the operator that produced its offer. newID and now fall
// back to defaults if nil.
func NewBookingService(operators []ports.OperatorPort, repo ports.BookingRepository, logger *slog.Logger, now func() time.Time, newID func() string) *BookingService {
	index := make(map[string]ports.OperatorPort, len(operators))
	for _, op := range operators {
		index[op.Name()] = op
	}
	if newID == nil {
		newID = func() string { return fmt.Sprintf("bk-%d", time.Now().UnixNano()) }
	}
	return &BookingService{
		operators: index,
		repo:      repo,
		logger:    loggerOr(logger),
		now:       clockOr(now),
		newID:     newID,
	}
}

// Book creates a pending booking, reconciles the offer with its operator, and
// either confirms it or records why it failed — persisting either outcome. It
// returns a transport error only for an unknown operator or a persistence fault.
func (s *BookingService) Book(ctx context.Context, req ports.BookRequest) (*domain.Booking, error) {
	op, ok := s.operators[req.Offer.Operator]
	if !ok {
		return nil, fmt.Errorf("book: %w: %q", ErrUnknownOperator, req.Offer.Operator)
	}

	booking, err := domain.NewBooking(s.newID(), req.Offer, req.Passengers, s.now())
	if err != nil {
		return nil, fmt.Errorf("book: %w", err)
	}

	rec, err := op.Reconcile(ctx, req.Offer)
	if err != nil {
		return s.failAndSave(ctx, booking, "reconciliation failed: "+err.Error())
	}
	if !rec.Available {
		return s.failAndSave(ctx, booking, "offer no longer available")
	}
	if rec.Offer.Price != req.Offer.Price {
		return s.failAndSave(ctx, booking,
			fmt.Sprintf("price changed: was %s, now %s", req.Offer.Price, rec.Offer.Price))
	}

	ref, err := op.Book(ctx, rec.Offer, req.Passengers)
	if err != nil {
		return s.failAndSave(ctx, booking, "operator booking failed: "+err.Error())
	}

	if err := booking.Confirm(ref, s.now()); err != nil {
		return nil, fmt.Errorf("book: confirm: %w", err)
	}
	if err := s.repo.Save(ctx, booking); err != nil {
		return nil, fmt.Errorf("book: persist confirmed booking: %w", err)
	}
	s.logger.InfoContext(ctx, "booking confirmed",
		"id", booking.ID(), "operator", op.Name(), "ref", ref)
	return booking, nil
}

// Get returns the current state of a booking by id.
func (s *BookingService) Get(ctx context.Context, id string) (*domain.Booking, error) {
	booking, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get booking %q: %w", id, err)
	}
	return booking, nil
}

func (s *BookingService) failAndSave(ctx context.Context, b *domain.Booking, reason string) (*domain.Booking, error) {
	if err := b.Fail(reason, s.now()); err != nil {
		return nil, fmt.Errorf("book: mark failed: %w", err)
	}
	if err := s.repo.Save(ctx, b); err != nil {
		return nil, fmt.Errorf("book: persist failed booking: %w", err)
	}
	s.logger.WarnContext(ctx, "booking failed", "id", b.ID(), "reason", reason)
	return b, nil
}
