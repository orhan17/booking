package http

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/orhan17/booking-aggregator/internal/app"
	"github.com/orhan17/booking-aggregator/internal/domain"
	"github.com/orhan17/booking-aggregator/internal/ports"
)

// itOperator is a minimal OperatorPort for wiring the real SearchService end to
// end: it can be healthy, erroring, or slow (honouring context cancellation).
type itOperator struct {
	name   string
	offers []domain.Offer
	err    error
	delay  time.Duration
}

func (o *itOperator) Name() string { return o.name }

func (o *itOperator) Search(ctx context.Context, _ domain.SearchCriteria) ([]domain.Offer, error) {
	if o.delay > 0 {
		select {
		case <-time.After(o.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if o.err != nil {
		return nil, o.err
	}
	return o.offers, nil
}

func (o *itOperator) Reconcile(context.Context, domain.Offer) (ports.ReconcileResult, error) {
	return ports.ReconcileResult{}, nil
}

func (o *itOperator) Book(context.Context, domain.Offer, int) (string, error) {
	return "", nil
}

// lockedBuffer is a concurrency-safe log sink; operator goroutines log failures
// in parallel during the fan-out.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func itOffer(id, operator string, priceMinor int64) domain.Offer {
	return domain.Offer{
		ID:            id,
		Operator:      operator,
		Origin:        "IST",
		Destination:   "AYT",
		DepartureDate: testClock.Add(48 * time.Hour),
		Passengers:    2,
		Price:         domain.Money{AmountMinor: priceMinor, Currency: "EUR"},
	}
}

// TestSearch_GracefulDegradation_EndToEnd wires the real SearchService behind
// the HTTP router with three operators: healthy, erroring, and slow-past-the-
// timeout. POST /search must return 200 with only the healthy operator's offers,
// and both failures must be logged.
func TestSearch_GracefulDegradation_EndToEnd(t *testing.T) {
	var logs lockedBuffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	healthy := &itOperator{name: "anex", offers: []domain.Offer{itOffer("anex-1", "anex", 39900)}}
	erroring := &itOperator{name: "coral", err: errors.New("upstream 503")}
	slow := &itOperator{
		name:   "sunmar",
		offers: []domain.Offer{itOffer("sunmar-1", "sunmar", 30000)},
		delay:  500 * time.Millisecond, // far beyond the 50ms budget below
	}

	now := func() time.Time { return testClock }
	searchSvc := app.NewSearchService(
		[]ports.OperatorPort{healthy, erroring, slow},
		logger,
		50*time.Millisecond, // overall timeout
		3,                   // bounded concurrency
		now,
	)
	router := NewRouter(searchSvc, stubBooking{}, logger)

	body := `{"origin":"IST","destination":"AYT","departure_date":"2026-07-08T00:00:00Z","passengers":2}`

	start := time.Now()
	rec := do(t, router, http.MethodPost, "/search", body)
	elapsed := time.Since(start)

	// The healthy operator's result comes back with a 200 despite the others.
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var resp searchResponse
	mustDecode(t, rec, &resp)
	if len(resp.Offers) != 1 || resp.Offers[0].ID != "anex-1" {
		t.Fatalf("offers = %+v, want exactly [anex-1]", resp.Offers)
	}

	// The request must not wait for the slow operator.
	if elapsed > 300*time.Millisecond {
		t.Fatalf("request took %v; slow operator was not dropped at the timeout", elapsed)
	}

	// Both failures must be logged (the erroring and the timed-out operators).
	logged := logs.String()
	dropCount := strings.Count(logged, "operator search failed, dropping")
	if dropCount != 2 {
		t.Fatalf("logged %d drops, want 2; logs:\n%s", dropCount, logged)
	}
	for _, name := range []string{"coral", "sunmar"} {
		if !strings.Contains(logged, name) {
			t.Fatalf("expected failing operator %q in logs; logs:\n%s", name, logged)
		}
	}
}
