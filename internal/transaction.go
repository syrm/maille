package internal

import (
	"time"
)

type Transaction struct {
	ID        uint64
	Date      time.Time
	Payee     string `expr:"payee"`
	Narration *string
	Postings  []Posting `expr:"postings"`
}

type Posting struct {
	ID       uint64
	Account  string  `expr:"account"`
	Amount   float64 `expr:"amount"`
	Currency Currency
}

type Currency struct {
	ID   uint
	Name string
}

type AccountType string

const (
	AccountTypeAssets      AccountType = "Assets"
	AccountTypeIncome      AccountType = "Income"
	AccountTypeExpenses    AccountType = "Expenses"
	AccountTypeLiabilities AccountType = "Liabilities"
	AccountTypeEquity      AccountType = "Equity"
)

type Account struct {
	ID   uint64
	Type AccountType
	Name string
}
