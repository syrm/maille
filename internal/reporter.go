package internal

import (
	"context"
	"log/slog"

	"github.com/syrm/maille/internal/domain"
	"github.com/syrm/maille/internal/domain/api"
)

const recentTransactionsLimit = 6

type transactionStatsProvider interface {
	GetCheckingBalance(context.Context) (int64, error)
	GetRecentTransactions(context.Context, uint) ([]domain.RecentTransaction, error)
	GetRecentTransactionsAPI(context.Context, uint) ([]api.Transaction, error)
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

func (r Reporter) BalanceSummaryAPI(ctx context.Context) api.BalanceSummary {
	checkingBalance, err := r.TransactionStatsProvider.GetCheckingBalance(ctx)

	if err != nil {
		r.Logger.ErrorContext(ctx, "failed to get balance summary", slog.Any("error", err))
		return api.BalanceSummary{}
	}

	return api.BalanceSummary{
		TotalBalance:    checkingBalance,
		CheckingBalance: checkingBalance,
	}
}

func (r Reporter) RecentTransactionsAPI(ctx context.Context) []api.Transaction {
	transactions, err := r.TransactionStatsProvider.GetRecentTransactionsAPI(ctx, recentTransactionsLimit)

	if err != nil {
		r.Logger.ErrorContext(ctx, "failed to get recent transaction", slog.Any("error", err))
		return nil
	}

	return transactions
}
