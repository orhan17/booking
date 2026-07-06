package domain

import (
	"fmt"
	"time"
)

// Status is the lifecycle state of a Booking.
type Status string

const (
	StatusPending   Status = "pending"
	StatusConfirmed Status = "confirmed"
	StatusFailed    Status = "failed"
)

var allowedTransitions = map[Status]map[Status]bool{
	StatusPending: {
		StatusConfirmed: true,
		StatusFailed:    true,
	},
	StatusConfirmed: {},
	StatusFailed:    {},
}

// IsTerminal reports whether the status admits no further transitions.
func (s Status) IsTerminal() bool {
	return len(allowedTransitions[s]) == 0
}

// Booking is the aggregate root for a reservation. Its state changes only
// through Confirm and Fail, which enforce the state machine.
type Booking struct {
	id              string
	offer           Offer
	passengers      int
	status          Status
	confirmationRef string
	failureReason   string
	createdAt       time.Time
	updatedAt       time.Time
}

// NewBooking creates a booking in the pending state.
func NewBooking(id string, offer Offer, passengers int, now time.Time) (*Booking, error) {
	if id == "" {
		return nil, fmt.Errorf("new booking: %w", ErrEmptyBookingID)
	}
	if passengers < 1 {
		return nil, fmt.Errorf("new booking: %w", ErrNoPassengers)
	}
	return &Booking{
		id:         id,
		offer:      offer,
		passengers: passengers,
		status:     StatusPending,
		createdAt:  now,
		updatedAt:  now,
	}, nil
}

// Confirm moves a pending booking to confirmed, recording the operator's
// reference. It returns ErrInvalidTransition if the booking is not pending.
func (b *Booking) Confirm(confirmationRef string, now time.Time) error {
	if err := b.transitionTo(StatusConfirmed, now); err != nil {
		return err
	}
	b.confirmationRef = confirmationRef
	return nil
}

// Fail moves a pending booking to failed, recording the reason. It returns
// ErrInvalidTransition if the booking is not pending.
func (b *Booking) Fail(reason string, now time.Time) error {
	if err := b.transitionTo(StatusFailed, now); err != nil {
		return err
	}
	b.failureReason = reason
	return nil
}

func (b *Booking) transitionTo(target Status, now time.Time) error {
	if !allowedTransitions[b.status][target] {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, b.status, target)
	}
	b.status = target
	b.updatedAt = now
	return nil
}

func (b *Booking) ID() string              { return b.id }
func (b *Booking) Offer() Offer            { return b.offer }
func (b *Booking) Passengers() int         { return b.passengers }
func (b *Booking) Status() Status          { return b.status }
func (b *Booking) ConfirmationRef() string { return b.confirmationRef }
func (b *Booking) FailureReason() string   { return b.failureReason }
func (b *Booking) CreatedAt() time.Time    { return b.createdAt }
func (b *Booking) UpdatedAt() time.Time    { return b.updatedAt }
