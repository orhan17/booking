package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/orhan17/booking-aggregator/internal/app"
	"github.com/orhan17/booking-aggregator/internal/domain"
	"github.com/orhan17/booking-aggregator/internal/ports"
)

type stubSearch struct {
	offers []domain.Offer
	err    error
}

func (s stubSearch) Search(_ context.Context, _ domain.SearchCriteria) ([]domain.Offer, error) {
	return s.offers, s.err
}

type stubBooking struct {
	bookFn func(context.Context, ports.BookRequest) (*domain.Booking, error)
	getFn  func(context.Context, string) (*domain.Booking, error)
}

func (s stubBooking) Book(ctx context.Context, req ports.BookRequest) (*domain.Booking, error) {
	return s.bookFn(ctx, req)
}

func (s stubBooking) Get(ctx context.Context, id string) (*domain.Booking, error) {
	return s.getFn(ctx, id)
}

var testClock = time.Date(2026, time.July, 6, 12, 0, 0, 0, time.UTC)

func testOffer() domain.Offer {
	return domain.Offer{
		ID:            "anex-1",
		Operator:      "anex",
		Origin:        "IST",
		Destination:   "AYT",
		DepartureDate: testClock.Add(48 * time.Hour),
		Passengers:    2,
		Price:         domain.Money{AmountMinor: 39900, Currency: "EUR"},
	}
}

func confirmedBooking(t *testing.T) *domain.Booking {
	t.Helper()
	b, err := domain.NewBooking("bk-1", testOffer(), 2, testClock)
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}
	if err := b.Confirm("ANEX-000123", testClock); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	return b
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, reader)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHandleSearch_OK(t *testing.T) {
	offers := []domain.Offer{testOffer()}
	router := NewRouter(stubSearch{offers: offers}, stubBooking{}, nil)

	body := `{"origin":"IST","destination":"AYT","departure_date":"2026-07-08T00:00:00Z","passengers":2}`
	rec := do(t, router, http.MethodPost, "/search", body)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var resp searchResponse
	mustDecode(t, rec, &resp)
	if len(resp.Offers) != 1 || resp.Offers[0].ID != "anex-1" {
		t.Fatalf("offers = %+v, want one offer anex-1", resp.Offers)
	}
	if resp.Offers[0].Price.Display == "" {
		t.Fatalf("price display not populated: %+v", resp.Offers[0].Price)
	}
}

func TestHandleSearch_BadJSON(t *testing.T) {
	router := NewRouter(stubSearch{}, stubBooking{}, nil)
	rec := do(t, router, http.MethodPost, "/search", `{"origin":`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleSearch_ValidationError(t *testing.T) {
	// The use case rejects the criteria; the handler maps it to 400.
	router := NewRouter(stubSearch{err: fmt.Errorf("search: %w", domain.ErrEmptyOrigin)}, stubBooking{}, nil)
	body := `{"origin":"","destination":"AYT","departure_date":"2026-07-08T00:00:00Z","passengers":2}`
	rec := do(t, router, http.MethodPost, "/search", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleCreateBooking_Confirmed(t *testing.T) {
	booking := confirmedBooking(t)
	stub := stubBooking{
		bookFn: func(_ context.Context, _ ports.BookRequest) (*domain.Booking, error) {
			return booking, nil
		},
	}
	router := NewRouter(stubSearch{}, stub, nil)

	body := `{"offer":{"id":"anex-1","operator":"anex","origin":"IST","destination":"AYT","departure_date":"2026-07-08T00:00:00Z","passengers":2,"price":{"amount_minor":39900,"currency":"EUR"}},"passengers":2}`
	rec := do(t, router, http.MethodPost, "/bookings", body)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != "/bookings/bk-1" {
		t.Fatalf("Location = %q, want /bookings/bk-1", loc)
	}
	var resp bookingResponse
	mustDecode(t, rec, &resp)
	if resp.Status != "confirmed" || resp.ConfirmationRef != "ANEX-000123" {
		t.Fatalf("resp = %+v, want confirmed with ref", resp)
	}
}

func TestHandleCreateBooking_FailedIsStill201(t *testing.T) {
	// A business failure (e.g. price changed) is a persisted resource, not an
	// HTTP error: 201 with status "failed".
	b, _ := domain.NewBooking("bk-2", testOffer(), 2, testClock)
	_ = b.Fail("price changed", testClock)
	stub := stubBooking{
		bookFn: func(_ context.Context, _ ports.BookRequest) (*domain.Booking, error) { return b, nil },
	}
	router := NewRouter(stubSearch{}, stub, nil)

	body := `{"offer":{"id":"anex-1","operator":"anex","origin":"IST","destination":"AYT","departure_date":"2026-07-08T00:00:00Z","passengers":2,"price":{"amount_minor":39900,"currency":"EUR"}},"passengers":2}`
	rec := do(t, router, http.MethodPost, "/bookings", body)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	var resp bookingResponse
	mustDecode(t, rec, &resp)
	if resp.Status != "failed" || resp.FailureReason != "price changed" {
		t.Fatalf("resp = %+v, want failed with reason", resp)
	}
}

func TestHandleCreateBooking_UnknownOperator(t *testing.T) {
	stub := stubBooking{
		bookFn: func(_ context.Context, _ ports.BookRequest) (*domain.Booking, error) {
			return nil, fmt.Errorf("book: %w: %q", app.ErrUnknownOperator, "nope")
		},
	}
	router := NewRouter(stubSearch{}, stub, nil)

	body := `{"offer":{"id":"x","operator":"nope","origin":"IST","destination":"AYT","departure_date":"2026-07-08T00:00:00Z","passengers":1,"price":{"amount_minor":1000,"currency":"EUR"}},"passengers":1}`
	rec := do(t, router, http.MethodPost, "/bookings", body)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleGetBooking_Found(t *testing.T) {
	booking := confirmedBooking(t)
	stub := stubBooking{
		getFn: func(_ context.Context, id string) (*domain.Booking, error) {
			if id != "bk-1" {
				t.Fatalf("got id %q, want bk-1", id)
			}
			return booking, nil
		},
	}
	router := NewRouter(stubSearch{}, stub, nil)

	rec := do(t, router, http.MethodGet, "/bookings/bk-1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var resp bookingResponse
	mustDecode(t, rec, &resp)
	if resp.ID != "bk-1" {
		t.Fatalf("resp.ID = %q, want bk-1", resp.ID)
	}
}

func TestHandleGetBooking_NotFound(t *testing.T) {
	stub := stubBooking{
		getFn: func(_ context.Context, id string) (*domain.Booking, error) {
			return nil, fmt.Errorf("get booking %q: %w", id, domain.ErrBookingNotFound)
		},
	}
	router := NewRouter(stubSearch{}, stub, nil)

	rec := do(t, router, http.MethodGet, "/bookings/missing", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestRouting_MethodAndPath(t *testing.T) {
	router := NewRouter(stubSearch{}, stubBooking{}, nil)

	// Wrong method on a known path → 405 from the mux.
	rec := do(t, router, http.MethodGet, "/search", "")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET /search status = %d, want 405", rec.Code)
	}

	// Health endpoint.
	rec = do(t, router, http.MethodGet, "/healthz", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want 200", rec.Code)
	}
}

func mustDecode(t *testing.T, rec *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	if err := json.NewDecoder(rec.Body).Decode(dst); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
