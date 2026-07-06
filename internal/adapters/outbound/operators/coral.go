package operators

import "time"

// NewCoral returns the "coral" operator: moderate latency and pricier but
// reliable, nudging its price up at reconciliation to exercise price-change
// handling.
func NewCoral() *MockOperator {
	return NewMockOperator(Config{
		Name:            "coral",
		Latency:         120 * time.Millisecond,
		LatencyJitter:   60 * time.Millisecond,
		BasePriceMinor:  42900,
		PriceStepMinor:  6000,
		Currency:        "EUR",
		OfferCount:      2,
		PriceDeltaMinor: 2000,
		Seed:            2,
	})
}
