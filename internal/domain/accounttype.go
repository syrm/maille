package domain

import "slices"

type AccountType string

const (
	AccountTypeAssets      AccountType = "Assets"
	AccountTypeIncome      AccountType = "Income"
	AccountTypeExpenses    AccountType = "Expenses"
	AccountTypeLiabilities AccountType = "Liabilities"
	AccountTypeEquity      AccountType = "Equity"
)

func accountTypeList() []AccountType {
	return []AccountType{
		AccountTypeAssets,
		AccountTypeIncome,
		AccountTypeExpenses,
		AccountTypeLiabilities,
		AccountTypeEquity,
	}
}

func (accountType AccountType) IsValid() bool {
	return slices.Contains(accountTypeList(), accountType)
}
