package operators

import "time"

// NewSunmar returns the "sunmar" operator: slow and flaky, so it is the one most
// likely to be dropped by the fan-out, demonstrating graceful degradation.
func NewSunmar() *MockOperator {
	return NewMockOperator(Config{
		Name:           "sunmar",
		Latency:        250 * time.Millisecond,
		LatencyJitter:  150 * time.Millisecond,
		FailureRate:    0.25,
		BasePriceMinor: 37900,
		PriceStepMinor: 3500,
		Currency:       "EUR",
		OfferCount:     4,
		Seed:           3,
	})
}
