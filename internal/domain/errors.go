package domain

import "errors"

// Sentinel domain errors, checked with errors.Is and wrapped with %w.
var (
	ErrEmptyOrigin           = errors.New("origin must not be empty")
	ErrEmptyDestination      = errors.New("destination must not be empty")
	ErrSameOriginDestination = errors.New("origin and destination must differ")
	ErrMissingDepartureDate  = errors.New("departure date is required")
	ErrDepartureInPast       = errors.New("departure date must be in the future")
	ErrReturnBeforeDeparture = errors.New("return date must be on or after departure date")
	ErrNoPassengers          = errors.New("at least one passenger is required")
	ErrTooManyPassengers     = errors.New("passenger count exceeds the maximum")

	ErrEmptyBookingID    = errors.New("booking id must not be empty")
	ErrInvalidTransition = errors.New("invalid booking state transition")

	ErrBookingNotFound = errors.New("booking not found")
)
