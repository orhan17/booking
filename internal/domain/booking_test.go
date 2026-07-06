package domain

import (
	"errors"
	"testing"
	"time"
)

func testOffer() Offer {
	depart := time.Date(2026, time.July, 8, 8, 0, 0, 0, time.UTC)
	return Offer{
		ID:            "offer-1",
		Operator:      "anex",
		Origin:        "IST",
		Destination:   "AYT",
		DepartureDate: depart,
		ReturnDate:    depart.Add(7 * 24 * time.Hour),
		Passengers:    2,
		Price:         Money{AmountMinor: 49900, Currency: "EUR"},
	}
}

func TestNewBooking(t *testing.T) {
	now := time.Date(2026, time.July, 6, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		id         string
		passengers int
		wantErr    error
	}{
		{name: "valid", id: "bk-1", passengers: 2},
		{name: "empty id", id: "", passengers: 2, wantErr: ErrEmptyBookingID},
		{name: "zero passengers", id: "bk-1", passengers: 0, wantErr: ErrNoPassengers},
		{name: "negative passengers", id: "bk-1", passengers: -3, wantErr: ErrNoPassengers},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := NewBooking(tt.id, testOffer(), tt.passengers, now)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("NewBooking() err = %v, want errors.Is(..., %v)", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewBooking() unexpected err = %v", err)
			}
			if b.Status() != StatusPending {
				t.Fatalf("new booking status = %q, want %q", b.Status(), StatusPending)
			}
			if !b.CreatedAt().Equal(now) || !b.UpdatedAt().Equal(now) {
				t.Fatalf("timestamps = (created %v, updated %v), want both %v", b.CreatedAt(), b.UpdatedAt(), now)
			}
		})
	}
}

// TestBooking_Transitions exercises the state machine from every starting state
// against both terminal transitions, asserting valid moves succeed and invalid
// moves are rejected with ErrInvalidTransition and leave the state unchanged.
func TestBooking_Transitions(t *testing.T) {
	now := time.Date(2026, time.July, 6, 12, 0, 0, 0, time.UTC)
	later := now.Add(time.Hour)

	// action applies a transition to a booking.
	confirm := func(b *Booking) error { return b.Confirm("ref-123", later) }
	fail := func(b *Booking) error { return b.Fail("operator rejected", later) }

	// start moves a fresh pending booking into the desired starting state.
	startPending := func(_ *Booking) {}
	startConfirmed := func(b *Booking) {
		if err := b.Confirm("ref-0", now); err != nil {
			t.Fatalf("setup Confirm failed: %v", err)
		}
	}
	startFailed := func(b *Booking) {
		if err := b.Fail("setup", now); err != nil {
			t.Fatalf("setup Fail failed: %v", err)
		}
	}

	tests := []struct {
		name      string
		start     func(*Booking)
		fromState Status
		action    func(*Booking) error
		wantState Status
		wantErr   error
	}{
		{"pending -> confirmed", startPending, StatusPending, confirm, StatusConfirmed, nil},
		{"pending -> failed", startPending, StatusPending, fail, StatusFailed, nil},
		{"confirmed -> confirmed rejected", startConfirmed, StatusConfirmed, confirm, StatusConfirmed, ErrInvalidTransition},
		{"confirmed -> failed rejected", startConfirmed, StatusConfirmed, fail, StatusConfirmed, ErrInvalidTransition},
		{"failed -> confirmed rejected", startFailed, StatusFailed, confirm, StatusFailed, ErrInvalidTransition},
		{"failed -> failed rejected", startFailed, StatusFailed, fail, StatusFailed, ErrInvalidTransition},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := NewBooking("bk-1", testOffer(), 2, now)
			if err != nil {
				t.Fatalf("NewBooking() err = %v", err)
			}
			tt.start(b)
			if b.Status() != tt.fromState {
				t.Fatalf("setup landed in %q, want %q", b.Status(), tt.fromState)
			}

			err = tt.action(b)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("transition err = %v, want errors.Is(..., %v)", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("transition unexpected err = %v", err)
			}
			if b.Status() != tt.wantState {
				t.Fatalf("status after transition = %q, want %q", b.Status(), tt.wantState)
			}
		})
	}
}

func TestBooking_Confirm_RecordsReference(t *testing.T) {
	now := time.Date(2026, time.July, 6, 12, 0, 0, 0, time.UTC)
	later := now.Add(time.Hour)

	b, err := NewBooking("bk-1", testOffer(), 2, now)
	if err != nil {
		t.Fatalf("NewBooking() err = %v", err)
	}
	if err := b.Confirm("PNR-XYZ", later); err != nil {
		t.Fatalf("Confirm() err = %v", err)
	}
	if b.ConfirmationRef() != "PNR-XYZ" {
		t.Fatalf("ConfirmationRef() = %q, want %q", b.ConfirmationRef(), "PNR-XYZ")
	}
	if !b.UpdatedAt().Equal(later) {
		t.Fatalf("UpdatedAt() = %v, want %v", b.UpdatedAt(), later)
	}
}

func TestBooking_Fail_RecordsReason(t *testing.T) {
	now := time.Date(2026, time.July, 6, 12, 0, 0, 0, time.UTC)
	later := now.Add(time.Hour)

	b, err := NewBooking("bk-1", testOffer(), 2, now)
	if err != nil {
		t.Fatalf("NewBooking() err = %v", err)
	}
	if err := b.Fail("price changed", later); err != nil {
		t.Fatalf("Fail() err = %v", err)
	}
	if b.FailureReason() != "price changed" {
		t.Fatalf("FailureReason() = %q, want %q", b.FailureReason(), "price changed")
	}
}

func TestStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status Status
		want   bool
	}{
		{StatusPending, false},
		{StatusConfirmed, true},
		{StatusFailed, true},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsTerminal(); got != tt.want {
				t.Fatalf("%q.IsTerminal() = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
