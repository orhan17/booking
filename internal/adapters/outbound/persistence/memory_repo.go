// Package persistence contains outbound storage adapters. MemoryBookingRepository
// is the default BookingRepository, intended to be swapped for a Postgres-backed
// one later without touching the app or domain.
package persistence

import (
	"context"
	"sync"

	"github.com/orhan17/booking-aggregator/internal/domain"
	"github.com/orhan17/booking-aggregator/internal/ports"
)

// MemoryBookingRepository is a concurrency-safe in-memory BookingRepository. It
// stores and returns copies so callers can't mutate persisted state via an
// aliased pointer.
type MemoryBookingRepository struct {
	mu       sync.RWMutex
	bookings map[string]domain.Booking
}

var _ ports.BookingRepository = (*MemoryBookingRepository)(nil)

// NewMemoryBookingRepository returns an empty in-memory repository.
func NewMemoryBookingRepository() *MemoryBookingRepository {
	return &MemoryBookingRepository{bookings: make(map[string]domain.Booking)}
}

// Save inserts or replaces the booking, keyed by its id.
func (r *MemoryBookingRepository) Save(_ context.Context, booking *domain.Booking) error {
	if booking == nil {
		return domain.ErrEmptyBookingID
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bookings[booking.ID()] = *booking
	return nil
}

// FindByID returns a copy of the stored booking, or domain.ErrBookingNotFound.
func (r *MemoryBookingRepository) FindByID(_ context.Context, id string) (*domain.Booking, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.bookings[id]
	if !ok {
		return nil, domain.ErrBookingNotFound
	}
	return &b, nil
}
