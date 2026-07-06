package domain

import (
	"fmt"
	"time"
)

// Money is a value object holding a price in integer minor units (e.g. cents)
// to avoid floating-point rounding on money.
type Money struct {
	AmountMinor int64
	Currency    string
}

// String renders the amount with two decimal places and the currency code.
func (m Money) String() string {
	return fmt.Sprintf("%d.%02d %s", m.AmountMinor/100, abs64(m.AmountMinor%100), m.Currency)
}

// LessThan reports whether m is cheaper than other, assuming a shared currency.
func (m Money) LessThan(other Money) bool {
	return m.AmountMinor < other.AmountMinor
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

// Offer is an immutable, priced result returned by an operator for a search.
type Offer struct {
	ID            string
	Operator      string
	Origin        string
	Destination   string
	DepartureDate time.Time
	ReturnDate    time.Time
	Passengers    int
	Price         Money
}

// CheaperThan reports whether o costs less than other.
func (o Offer) CheaperThan(other Offer) bool {
	return o.Price.LessThan(other.Price)
}
