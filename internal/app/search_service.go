package app

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/orhan17/booking-aggregator/internal/domain"
	"github.com/orhan17/booking-aggregator/internal/ports"
)

// SearchService fans a search out to every operator concurrently and returns
// the merged, price-sorted offers. A slow or failing operator is logged and
// dropped, never failing the request.
type SearchService struct {
	operators []ports.OperatorPort
	logger    *slog.Logger
	timeout   time.Duration
	limit     int
	now       func() time.Time
}

var _ ports.SearchUseCase = (*SearchService)(nil)

// NewSearchService wires the search use case. timeout bounds the whole fan-out
// (default 3s if <= 0); limit bounds concurrency (default: one slot per
// operator). logger and now fall back to sensible defaults if nil.
func NewSearchService(operators []ports.OperatorPort, logger *slog.Logger, timeout time.Duration, limit int, now func() time.Time) *SearchService {
	if timeout <= 0 {
		timeout = defaultSearchTimeout
	}
	if limit <= 0 {
		limit = len(operators)
	}
	if limit < 1 {
		limit = 1
	}
	return &SearchService{
		operators: operators,
		logger:    loggerOr(logger),
		timeout:   timeout,
		limit:     limit,
		now:       clockOr(now),
	}
}

// Search validates the criteria, then queries all operators under a shared
// timeout with bounded concurrency, merging whatever comes back in time.
func (s *SearchService) Search(ctx context.Context, criteria domain.SearchCriteria) ([]domain.Offer, error) {
	if err := criteria.Validate(s.now()); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// Goroutines never return an error, so the group's context is cancelled only
	// by the timeout or the caller — one operator's failure never cancels others.
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(s.limit)

	var (
		mu     sync.Mutex
		offers []domain.Offer
	)

	for _, op := range s.operators {
		g.Go(func() error {
			res, err := op.Search(ctx, criteria)
			if err != nil {
				s.logger.WarnContext(ctx, "operator search failed, dropping",
					"operator", op.Name(), "error", err)
				return nil
			}
			mu.Lock()
			offers = append(offers, res...)
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait()

	sort.SliceStable(offers, func(i, j int) bool {
		return offers[i].CheaperThan(offers[j])
	})
	return offers, nil
}
