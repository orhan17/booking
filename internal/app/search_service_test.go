package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/orhan17/booking-aggregator/internal/domain"
	"github.com/orhan17/booking-aggregator/internal/ports"
)

func validCriteria(now func() time.Time) domain.SearchCriteria {
	return domain.SearchCriteria{
		Origin:        "IST",
		Destination:   "AYT",
		DepartureDate: now().Add(48 * time.Hour),
		Passengers:    2,
	}
}

func offer(id, operator string, priceMinor int64) domain.Offer {
	return domain.Offer{
		ID:       id,
		Operator: operator,
		Price:    domain.Money{AmountMinor: priceMinor, Currency: "EUR"},
	}
}

// TestSearchService_GracefulDegradation is the core resilience test: a fast
// operator, a slow operator that exceeds the timeout, and an erroring operator
// run together. Only the fast operator's results should come back, the request
// must not error, and the slow/erroring operators must be dropped.
func TestSearchService_GracefulDegradation(t *testing.T) {
	now := fixedClock()

	fastCheap := &mockOperator{name: "coral", offers: []domain.Offer{offer("c1", "coral", 20000)}}
	fastPricey := &mockOperator{name: "anex", offers: []domain.Offer{offer("a1", "anex", 50000)}}
	slow := &mockOperator{
		name:        "sunmar",
		offers:      []domain.Offer{offer("s1", "sunmar", 10000)},
		searchDelay: 500 * time.Millisecond, // far beyond the 50ms budget
	}
	erroring := &mockOperator{name: "tez", searchErr: errors.New("upstream 500")}

	svc := NewSearchService(
		[]ports.OperatorPort{fastPricey, slow, erroring, fastCheap},
		testLogger(),
		50*time.Millisecond, // overall timeout
		2,                   // bounded concurrency
		now,
	)

	start := time.Now()
	offers, err := svc.Search(context.Background(), validCriteria(now))
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Search() err = %v, want nil (graceful degradation)", err)
	}
	if len(offers) != 2 {
		t.Fatalf("got %d offers, want 2 (only the fast operators): %+v", len(offers), offers)
	}
	// Sorted ascending by price: coral (200) before anex (500).
	if offers[0].ID != "c1" || offers[1].ID != "a1" {
		t.Fatalf("offers not price-sorted: got [%s, %s], want [c1, a1]", offers[0].ID, offers[1].ID)
	}
	// The request must return around the timeout, not wait for the slow operator.
	if elapsed > 300*time.Millisecond {
		t.Fatalf("Search took %v; slow operator was not dropped at the timeout", elapsed)
	}
}

func TestSearchService_AllOperatorsFail(t *testing.T) {
	now := fixedClock()
	op1 := &mockOperator{name: "a", searchErr: errors.New("boom")}
	op2 := &mockOperator{name: "b", searchDelay: 200 * time.Millisecond}

	svc := NewSearchService([]ports.OperatorPort{op1, op2}, testLogger(), 30*time.Millisecond, 2, now)

	offers, err := svc.Search(context.Background(), validCriteria(now))
	if err != nil {
		t.Fatalf("Search() err = %v, want nil", err)
	}
	if len(offers) != 0 {
		t.Fatalf("got %d offers, want 0", len(offers))
	}
}

func TestSearchService_MergesAndSortsAllFast(t *testing.T) {
	now := fixedClock()
	op1 := &mockOperator{name: "anex", offers: []domain.Offer{offer("a1", "anex", 50000), offer("a2", "anex", 15000)}}
	op2 := &mockOperator{name: "coral", offers: []domain.Offer{offer("c1", "coral", 30000)}}

	svc := NewSearchService([]ports.OperatorPort{op1, op2}, testLogger(), time.Second, 0, now)

	offers, err := svc.Search(context.Background(), validCriteria(now))
	if err != nil {
		t.Fatalf("Search() err = %v", err)
	}
	wantOrder := []string{"a2", "c1", "a1"} // 150, 300, 500
	if len(offers) != len(wantOrder) {
		t.Fatalf("got %d offers, want %d", len(offers), len(wantOrder))
	}
	for i, id := range wantOrder {
		if offers[i].ID != id {
			t.Fatalf("offers[%d].ID = %s, want %s (full: %+v)", i, offers[i].ID, id, offers)
		}
	}
}

func TestSearchService_InvalidCriteria(t *testing.T) {
	now := fixedClock()
	op := &mockOperator{name: "anex", offers: []domain.Offer{offer("a1", "anex", 1000)}}
	svc := NewSearchService([]ports.OperatorPort{op}, testLogger(), time.Second, 1, now)

	bad := validCriteria(now)
	bad.Origin = "" // fails validation

	offers, err := svc.Search(context.Background(), bad)
	if !errors.Is(err, domain.ErrEmptyOrigin) {
		t.Fatalf("Search() err = %v, want ErrEmptyOrigin", err)
	}
	if offers != nil {
		t.Fatalf("offers = %+v, want nil on validation failure", offers)
	}
}
