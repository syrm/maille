package domain

import (
	"time"

	"github.com/bojanz/currency"
)

type TransactionListFilter struct {
	Search        string
	CategoryID    uint64
	Uncategorized bool
	Limit         uint
	Offset        uint
}

type TransactionListItem struct {
	ID            uint64
	Date          time.Time
	Payee         string
	Classified    bool
	CategoryID    uint64
	Category      string
	CategoryAlias string
	CategoryIcon  string
	CategoryColor string
	BankAccount   string
	Amount        currency.Amount
}
