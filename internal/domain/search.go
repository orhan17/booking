package domain

import (
	"fmt"
	"time"
)

// MaxPassengers caps the number of passengers a single search may request.
const MaxPassengers = 9

// SearchCriteria is a validated value object describing a search request.
type SearchCriteria struct {
	Origin        string
	Destination   string
	DepartureDate time.Time
	ReturnDate    time.Time
	Passengers    int
}

// Validate checks the criteria against the domain rules, returning the first
// violation wrapped around a sentinel. now is injected for deterministic checks.
func (c SearchCriteria) Validate(now time.Time) error {
	if c.Origin == "" {
		return fmt.Errorf("validate search criteria: %w", ErrEmptyOrigin)
	}
	if c.Destination == "" {
		return fmt.Errorf("validate search criteria: %w", ErrEmptyDestination)
	}
	if c.Origin == c.Destination {
		return fmt.Errorf("validate search criteria: %w", ErrSameOriginDestination)
	}
	if c.DepartureDate.IsZero() {
		return fmt.Errorf("validate search criteria: %w", ErrMissingDepartureDate)
	}
	if c.DepartureDate.Before(now) {
		return fmt.Errorf("validate search criteria: %w", ErrDepartureInPast)
	}
	if !c.ReturnDate.IsZero() && c.ReturnDate.Before(c.DepartureDate) {
		return fmt.Errorf("validate search criteria: %w", ErrReturnBeforeDeparture)
	}
	if c.Passengers < 1 {
		return fmt.Errorf("validate search criteria: %w", ErrNoPassengers)
	}
	if c.Passengers > MaxPassengers {
		return fmt.Errorf("validate search criteria: %w", ErrTooManyPassengers)
	}
	return nil
}
