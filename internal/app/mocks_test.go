package app

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/orhan17/booking-aggregator/internal/domain"
	"github.com/orhan17/booking-aggregator/internal/ports"
)

// mockOperator is a configurable OperatorPort for tests: it can be fast, slow
// (past the service timeout), or erroring, and its reconcile/book behaviour is
// scriptable independently.
type mockOperator struct {
	name string

	// Search behaviour.
	offers      []domain.Offer
	searchErr   error
	searchDelay time.Duration

	// Reconcile behaviour.
	reconcile    ports.ReconcileResult
	reconcileErr error

	// Book behaviour.
	bookRef string
	bookErr error
}

func (m *mockOperator) Name() string { return m.name }

func (m *mockOperator) Search(ctx context.Context, _ domain.SearchCriteria) ([]domain.Offer, error) {
	if m.searchDelay > 0 {
		select {
		case <-time.After(m.searchDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.offers, nil
}

func (m *mockOperator) Reconcile(_ context.Context, _ domain.Offer) (ports.ReconcileResult, error) {
	if m.reconcileErr != nil {
		return ports.ReconcileResult{}, m.reconcileErr
	}
	return m.reconcile, nil
}

func (m *mockOperator) Book(_ context.Context, _ domain.Offer, _ int) (string, error) {
	if m.bookErr != nil {
		return "", m.bookErr
	}
	return m.bookRef, nil
}

// fakeRepo is an in-memory BookingRepository for tests.
type fakeRepo struct {
	mu    sync.Mutex
	saved map[string]*domain.Booking
	saves int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{saved: make(map[string]*domain.Booking)}
}

func (r *fakeRepo) Save(_ context.Context, b *domain.Booking) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saved[b.ID()] = b
	r.saves++
	return nil
}

func (r *fakeRepo) FindByID(_ context.Context, id string) (*domain.Booking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.saved[id]
	if !ok {
		return nil, domain.ErrBookingNotFound
	}
	return b, nil
}

// testLogger discards output so tests stay quiet.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fixedClock returns a deterministic clock function.
func fixedClock() func() time.Time {
	t := time.Date(2026, time.July, 6, 12, 0, 0, 0, time.UTC)
	return func() time.Time { return t }
}
