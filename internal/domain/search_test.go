package domain

import (
	"errors"
	"testing"
	"time"
)

func TestSearchCriteria_Validate(t *testing.T) {
	// Fixed reference time so "past" and "future" are deterministic.
	now := time.Date(2026, time.July, 6, 12, 0, 0, 0, time.UTC)
	depart := now.Add(48 * time.Hour)
	ret := depart.Add(7 * 24 * time.Hour)

	valid := SearchCriteria{
		Origin:        "IST",
		Destination:   "AYT",
		DepartureDate: depart,
		ReturnDate:    ret,
		Passengers:    2,
	}

	// mutate returns a copy of the valid criteria with f applied.
	mutate := func(f func(*SearchCriteria)) SearchCriteria {
		c := valid
		f(&c)
		return c
	}

	tests := []struct {
		name    string
		crit    SearchCriteria
		wantErr error // nil means "expect success"
	}{
		{
			name: "valid round trip",
			crit: valid,
		},
		{
			name: "valid one-way (no return date)",
			crit: mutate(func(c *SearchCriteria) { c.ReturnDate = time.Time{} }),
		},
		{
			name:    "empty origin",
			crit:    mutate(func(c *SearchCriteria) { c.Origin = "" }),
			wantErr: ErrEmptyOrigin,
		},
		{
			name:    "empty destination",
			crit:    mutate(func(c *SearchCriteria) { c.Destination = "" }),
			wantErr: ErrEmptyDestination,
		},
		{
			name:    "same origin and destination",
			crit:    mutate(func(c *SearchCriteria) { c.Destination = c.Origin }),
			wantErr: ErrSameOriginDestination,
		},
		{
			name:    "missing departure date",
			crit:    mutate(func(c *SearchCriteria) { c.DepartureDate = time.Time{} }),
			wantErr: ErrMissingDepartureDate,
		},
		{
			name:    "departure in the past",
			crit:    mutate(func(c *SearchCriteria) { c.DepartureDate = now.Add(-time.Hour) }),
			wantErr: ErrDepartureInPast,
		},
		{
			name:    "return before departure",
			crit:    mutate(func(c *SearchCriteria) { c.ReturnDate = depart.Add(-time.Hour) }),
			wantErr: ErrReturnBeforeDeparture,
		},
		{
			name: "return same day as departure is allowed",
			crit: mutate(func(c *SearchCriteria) { c.ReturnDate = depart }),
		},
		{
			name:    "zero passengers",
			crit:    mutate(func(c *SearchCriteria) { c.Passengers = 0 }),
			wantErr: ErrNoPassengers,
		},
		{
			name:    "negative passengers",
			crit:    mutate(func(c *SearchCriteria) { c.Passengers = -1 }),
			wantErr: ErrNoPassengers,
		},
		{
			name: "max passengers is allowed",
			crit: mutate(func(c *SearchCriteria) { c.Passengers = MaxPassengers }),
		},
		{
			name:    "too many passengers",
			crit:    mutate(func(c *SearchCriteria) { c.Passengers = MaxPassengers + 1 }),
			wantErr: ErrTooManyPassengers,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.crit.Validate(now)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() = %v, want errors.Is(..., %v)", err, tt.wantErr)
			}
		})
	}
}
