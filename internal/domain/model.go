package domain

import (
	"slices"
	"time"

	"github.com/expr-lang/expr/vm"
)

type Transaction struct {
	ID                    uint64
	ExternalID            string
	Date                  time.Time
	Payee                 string `expr:"payee"`
	Narration             *string
	Postings              []Posting `expr:"postings"`
	BankAccountExternalID string
}

type Posting struct {
	ID        uint64
	AccountID uint64
	Amount    float64 `expr:"amount"`
	Currency  Currency
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

type Account struct {
	ID   uint64
	Type AccountType
	Name string
}

type BankAccount struct {
	ID         uint64
	Name       string
	AccountID  uint64
	ExternalID string
}

type TransactionClassifierRule struct {
	ID      uint64
	Rule    string
	Account Account
	Program *vm.Program
}
