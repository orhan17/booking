package operators

import "time"

// NewAnex returns the "anex" operator: fast, reliable, and competitively priced.
func NewAnex() *MockOperator {
	return NewMockOperator(Config{
		Name:           "anex",
		Latency:        40 * time.Millisecond,
		LatencyJitter:  30 * time.Millisecond,
		BasePriceMinor: 39900,
		PriceStepMinor: 4000,
		Currency:       "EUR",
		OfferCount:     3,
		Seed:           1,
	})
}
