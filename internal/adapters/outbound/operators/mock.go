// Package operators contains outbound adapters implementing ports.OperatorPort,
// all backed by MockOperator — a configurable fake that simulates latency,
// failures, and price/availability drift so the app layer can be exercised
// end to end without real operator APIs.
package operators

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/orhan17/booking-aggregator/internal/domain"
	"github.com/orhan17/booking-aggregator/internal/ports"
)

var (
	errSearchFailed    = errors.New("operator search failed")
	errReconcileFailed = errors.New("operator reconcile failed")
	errBookFailed      = errors.New("operator booking failed")
)

// Config describes a MockOperator's simulated behaviour. The zero value yields a
// fast, reliable operator returning a single offer at BasePriceMinor.
type Config struct {
	Name string

	Latency       time.Duration
	LatencyJitter time.Duration

	FailureRate float64

	FailSearch    bool
	FailReconcile bool
	FailBook      bool

	Unavailable     bool
	PriceDeltaMinor int64

	BasePriceMinor int64
	PriceStepMinor int64
	Currency       string
	OfferCount     int

	Seed int64
}

// MockOperator is a configurable OperatorPort implementation.
type MockOperator struct {
	cfg Config

	mu  sync.Mutex
	rng *rand.Rand
}

var _ ports.OperatorPort = (*MockOperator)(nil)

// NewMockOperator builds a MockOperator from cfg.
func NewMockOperator(cfg Config) *MockOperator {
	return &MockOperator{
		cfg: cfg,
		rng: rand.New(rand.NewSource(cfg.Seed)), //nolint:gosec // not security-sensitive
	}
}

// Name returns the operator's name; it matches the Operator field on its offers.
func (m *MockOperator) Name() string { return m.cfg.Name }

// Search simulates latency, may inject a failure, then returns generated offers.
func (m *MockOperator) Search(ctx context.Context, criteria domain.SearchCriteria) ([]domain.Offer, error) {
	if err := m.simulate(ctx, m.cfg.Latency); err != nil {
		return nil, err
	}
	if m.cfg.FailSearch || m.rollFailure() {
		return nil, fmt.Errorf("%s: %w", m.cfg.Name, errSearchFailed)
	}
	return m.generateOffers(criteria), nil
}

// Reconcile re-validates the offer, optionally reporting unavailability or a
// shifted price to model state changing between search and booking.
func (m *MockOperator) Reconcile(ctx context.Context, offer domain.Offer) (ports.ReconcileResult, error) {
	if err := m.simulate(ctx, m.cfg.Latency/3); err != nil {
		return ports.ReconcileResult{}, err
	}
	if m.cfg.FailReconcile {
		return ports.ReconcileResult{}, fmt.Errorf("%s: %w", m.cfg.Name, errReconcileFailed)
	}
	if m.cfg.Unavailable {
		return ports.ReconcileResult{Available: false, Offer: offer}, nil
	}
	fresh := offer
	fresh.Price.AmountMinor += m.cfg.PriceDeltaMinor
	return ports.ReconcileResult{Available: true, Offer: fresh}, nil
}

// Book simulates reserving the offer and returns a confirmation reference.
func (m *MockOperator) Book(ctx context.Context, _ domain.Offer, _ int) (string, error) {
	if err := m.simulate(ctx, m.cfg.Latency/3); err != nil {
		return "", err
	}
	if m.cfg.FailBook {
		return "", fmt.Errorf("%s: %w", m.cfg.Name, errBookFailed)
	}
	return m.confirmationRef(), nil
}

// simulate sleeps for base + jitter, returning early if the context is done so a
// slow operator honours the caller's timeout.
func (m *MockOperator) simulate(ctx context.Context, base time.Duration) error {
	d := base + m.jitter()
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *MockOperator) jitter() time.Duration {
	if m.cfg.LatencyJitter <= 0 {
		return 0
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return time.Duration(m.rng.Int63n(int64(m.cfg.LatencyJitter)))
}

func (m *MockOperator) rollFailure() bool {
	if m.cfg.FailureRate <= 0 {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.rng.Float64() < m.cfg.FailureRate
}

func (m *MockOperator) confirmationRef() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return fmt.Sprintf("%s-%06d", strings.ToUpper(m.cfg.Name), m.rng.Intn(1_000_000))
}

func (m *MockOperator) generateOffers(c domain.SearchCriteria) []domain.Offer {
	n := m.cfg.OfferCount
	if n <= 0 {
		n = 1
	}
	currency := m.cfg.Currency
	if currency == "" {
		currency = "EUR"
	}

	offers := make([]domain.Offer, 0, n)
	for i := 0; i < n; i++ {
		price := m.cfg.BasePriceMinor + int64(i)*m.cfg.PriceStepMinor
		offers = append(offers, domain.Offer{
			ID:            fmt.Sprintf("%s-%d", m.cfg.Name, i+1),
			Operator:      m.cfg.Name,
			Origin:        c.Origin,
			Destination:   c.Destination,
			DepartureDate: c.DepartureDate,
			ReturnDate:    c.ReturnDate,
			Passengers:    c.Passengers,
			Price:         domain.Money{AmountMinor: price, Currency: currency},
		})
	}
	return offers
}
