package internal

import (
	"context"
	"log/slog"

	"github.com/syrm/maille/internal/domain"
)

const recentTransactionsLimit = 6

type transactionStatsProvider interface {
	GetCheckingBalance(context.Context) (int64, error)
	GetRecentTransactions(context.Context, uint) ([]domain.RecentTransaction, error)
	BalanceSummary(ctx context.Context) (domain.BalanceSummary, error)
}

type Reporter struct {
	TransactionStatsProvider transactionStatsProvider
	Logger                   *slog.Logger
}

func (r Reporter) BalanceSummary(ctx context.Context) domain.BalanceSummary {
	balanceSummary, err := r.TransactionStatsProvider.BalanceSummary(ctx)

	if err != nil {
		r.Logger.ErrorContext(ctx, "failed to get balance summary", slog.Any("error", err))
		return domain.BalanceSummary{}
	}

	return balanceSummary
}

func (r Reporter) RecentTransactions(ctx context.Context) []domain.RecentTransaction {
	transactions, err := r.TransactionStatsProvider.GetRecentTransactions(ctx, recentTransactionsLimit)

	if err != nil {
		r.Logger.ErrorContext(ctx, "failed to get recent transaction", slog.Any("error", err))
		return nil
	}

	return transactions
}
