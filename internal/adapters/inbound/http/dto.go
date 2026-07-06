package http

import (
	"time"

	"github.com/orhan17/booking-aggregator/internal/domain"
)

type moneyDTO struct {
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
	Display     string `json:"display,omitempty"`
}

type offerDTO struct {
	ID            string     `json:"id"`
	Operator      string     `json:"operator"`
	Origin        string     `json:"origin"`
	Destination   string     `json:"destination"`
	DepartureDate time.Time  `json:"departure_date"`
	ReturnDate    *time.Time `json:"return_date,omitempty"`
	Passengers    int        `json:"passengers"`
	Price         moneyDTO   `json:"price"`
}

func (o offerDTO) toDomain() domain.Offer {
	var ret time.Time
	if o.ReturnDate != nil {
		ret = *o.ReturnDate
	}
	return domain.Offer{
		ID:            o.ID,
		Operator:      o.Operator,
		Origin:        o.Origin,
		Destination:   o.Destination,
		DepartureDate: o.DepartureDate,
		ReturnDate:    ret,
		Passengers:    o.Passengers,
		Price:         domain.Money{AmountMinor: o.Price.AmountMinor, Currency: o.Price.Currency},
	}
}

func offerFromDomain(o domain.Offer) offerDTO {
	dto := offerDTO{
		ID:            o.ID,
		Operator:      o.Operator,
		Origin:        o.Origin,
		Destination:   o.Destination,
		DepartureDate: o.DepartureDate,
		Passengers:    o.Passengers,
		Price:         moneyDTO{AmountMinor: o.Price.AmountMinor, Currency: o.Price.Currency, Display: o.Price.String()},
	}
	if !o.ReturnDate.IsZero() {
		rt := o.ReturnDate
		dto.ReturnDate = &rt
	}
	return dto
}

func offersFromDomain(offers []domain.Offer) []offerDTO {
	out := make([]offerDTO, 0, len(offers))
	for _, o := range offers {
		out = append(out, offerFromDomain(o))
	}
	return out
}

type searchRequest struct {
	Origin        string     `json:"origin"`
	Destination   string     `json:"destination"`
	DepartureDate time.Time  `json:"departure_date"`
	ReturnDate    *time.Time `json:"return_date,omitempty"`
	Passengers    int        `json:"passengers"`
}

func (r searchRequest) toCriteria() domain.SearchCriteria {
	var ret time.Time
	if r.ReturnDate != nil {
		ret = *r.ReturnDate
	}
	return domain.SearchCriteria{
		Origin:        r.Origin,
		Destination:   r.Destination,
		DepartureDate: r.DepartureDate,
		ReturnDate:    ret,
		Passengers:    r.Passengers,
	}
}

type searchResponse struct {
	Offers []offerDTO `json:"offers"`
}

type bookingRequest struct {
	Offer      offerDTO `json:"offer"`
	Passengers int      `json:"passengers"`
}

type bookingResponse struct {
	ID              string    `json:"id"`
	Status          string    `json:"status"`
	Offer           offerDTO  `json:"offer"`
	Passengers      int       `json:"passengers"`
	ConfirmationRef string    `json:"confirmation_ref,omitempty"`
	FailureReason   string    `json:"failure_reason,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func bookingFromDomain(b *domain.Booking) bookingResponse {
	return bookingResponse{
		ID:              b.ID(),
		Status:          string(b.Status()),
		Offer:           offerFromDomain(b.Offer()),
		Passengers:      b.Passengers(),
		ConfirmationRef: b.ConfirmationRef(),
		FailureReason:   b.FailureReason(),
		CreatedAt:       b.CreatedAt(),
		UpdatedAt:       b.UpdatedAt(),
	}
}

type errorResponse struct {
	Error string `json:"error"`
}
