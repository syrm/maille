package api

type Dashboard struct {
	BalanceSummary     BalanceSummary
	RecentTransactions []Transaction
}
