package internal

import (
	"context"
	"log/slog"

	"github.com/syrm/maille/internal/domain"
)

type TransactionStatsProvider interface {
	GetTotal(context.Context) (int, error)
}

type Reporter struct {
	TransactionStatsProvider TransactionStatsProvider
	Logger                   *slog.Logger
}

func (r Reporter) Report(ctx context.Context) domain.Stats {
	total, err := r.TransactionStatsProvider.GetTotal(ctx)

	if err != nil {
		r.Logger.ErrorContext(ctx, "failed to get total", slog.Any("error", err))
		return domain.Stats{}
	}

	return domain.Stats{
		TotalTransaction: total,
	}
}
