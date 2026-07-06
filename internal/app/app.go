package app

import (
	"errors"
	"log/slog"
	"time"
)

// ErrUnknownOperator is returned by BookingService.Book when the selected offer
// names an operator the service was not configured with.
var ErrUnknownOperator = errors.New("unknown operator for offer")

const defaultSearchTimeout = 3 * time.Second

func clockOr(now func() time.Time) func() time.Time {
	if now != nil {
		return now
	}
	return time.Now
}

func loggerOr(logger *slog.Logger) *slog.Logger {
	if logger != nil {
		return logger
	}
	return slog.Default()
}
