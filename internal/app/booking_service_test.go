package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/orhan17/booking-aggregator/internal/domain"
	"github.com/orhan17/booking-aggregator/internal/ports"
)

func selectedOffer() domain.Offer {
	return offer("o1", "anex", 49900)
}

// bookingServiceWith builds a BookingService around a single mock operator and
// a fresh fake repo, with deterministic clock and id.
func bookingServiceWith(op *mockOperator) (*BookingService, *fakeRepo) {
	repo := newFakeRepo()
	svc := NewBookingService(
		[]ports.OperatorPort{op},
		repo,
		testLogger(),
		fixedClock(),
		func() string { return "bk-1" },
	)
	return svc, repo
}

func TestBookingService_Book(t *testing.T) {
	sel := selectedOffer()

	tests := []struct {
		name       string
		op         *mockOperator
		wantStatus domain.Status
		wantRef    string
		reasonHas  string // substring expected in FailureReason (when failed)
	}{
		{
			name: "happy path confirms",
			op: &mockOperator{
				name:      "anex",
				reconcile: ports.ReconcileResult{Available: true, Offer: sel},
				bookRef:   "PNR-777",
			},
			wantStatus: domain.StatusConfirmed,
			wantRef:    "PNR-777",
		},
		{
			name: "unavailable fails",
			op: &mockOperator{
				name:      "anex",
				reconcile: ports.ReconcileResult{Available: false, Offer: sel},
			},
			wantStatus: domain.StatusFailed,
			reasonHas:  "no longer available",
		},
		{
			name: "price change fails",
			op: &mockOperator{
				name:      "anex",
				reconcile: ports.ReconcileResult{Available: true, Offer: offer("o1", "anex", 59900)},
			},
			wantStatus: domain.StatusFailed,
			reasonHas:  "price changed",
		},
		{
			name: "reconcile error fails",
			op: &mockOperator{
				name:         "anex",
				reconcileErr: errors.New("operator timeout"),
			},
			wantStatus: domain.StatusFailed,
			reasonHas:  "reconciliation failed",
		},
		{
			name: "operator book error fails",
			op: &mockOperator{
				name:      "anex",
				reconcile: ports.ReconcileResult{Available: true, Offer: sel},
				bookErr:   errors.New("sold out"),
			},
			wantStatus: domain.StatusFailed,
			reasonHas:  "operator booking failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := bookingServiceWith(tt.op)

			b, err := svc.Book(context.Background(), ports.BookRequest{Offer: sel, Passengers: 2})
			if err != nil {
				t.Fatalf("Book() unexpected err = %v", err)
			}
			if b.Status() != tt.wantStatus {
				t.Fatalf("status = %q, want %q", b.Status(), tt.wantStatus)
			}
			if tt.wantRef != "" && b.ConfirmationRef() != tt.wantRef {
				t.Fatalf("confirmationRef = %q, want %q", b.ConfirmationRef(), tt.wantRef)
			}
			if tt.reasonHas != "" && !strings.Contains(b.FailureReason(), tt.reasonHas) {
				t.Fatalf("failureReason = %q, want to contain %q", b.FailureReason(), tt.reasonHas)
			}
			// Every outcome must be persisted so it is retrievable by id.
			if repo.saves != 1 {
				t.Fatalf("repo saves = %d, want 1", repo.saves)
			}
			got, err := svc.Get(context.Background(), b.ID())
			if err != nil {
				t.Fatalf("Get() err = %v", err)
			}
			if got.Status() != tt.wantStatus {
				t.Fatalf("persisted status = %q, want %q", got.Status(), tt.wantStatus)
			}
		})
	}
}

func TestBookingService_Book_UnknownOperator(t *testing.T) {
	op := &mockOperator{name: "anex", reconcile: ports.ReconcileResult{Available: true, Offer: selectedOffer()}}
	svc, repo := bookingServiceWith(op)

	req := ports.BookRequest{Offer: offer("x1", "unknown-operator", 1000), Passengers: 1}
	b, err := svc.Book(context.Background(), req)
	if !errors.Is(err, ErrUnknownOperator) {
		t.Fatalf("Book() err = %v, want ErrUnknownOperator", err)
	}
	if b != nil {
		t.Fatalf("booking = %+v, want nil", b)
	}
	if repo.saves != 0 {
		t.Fatalf("repo saves = %d, want 0", repo.saves)
	}
}

func TestBookingService_Get_NotFound(t *testing.T) {
	op := &mockOperator{name: "anex"}
	svc, _ := bookingServiceWith(op)

	_, err := svc.Get(context.Background(), "missing")
	if !errors.Is(err, domain.ErrBookingNotFound) {
		t.Fatalf("Get() err = %v, want ErrBookingNotFound", err)
	}
}
