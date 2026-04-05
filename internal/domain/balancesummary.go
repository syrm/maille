package domain

import "github.com/bojanz/currency"

type BalanceSummary struct {
	TotalBalance      currency.Amount
	CheckingBalance   currency.Amount
	SavingsBalance    currency.Amount
	InvestmentBalance currency.Amount
}
