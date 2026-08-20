package domain

import (
	"time"

	pkgcurrency "github.com/bojanz/currency"
)

type TransactionToClassify struct {
	ID        uint64
	Date      time.Time
	Payee     string `expr:"payee"`
	Narration string `expr:"narration"`
	PostingID uint64
	AccountID uint64
	Account   string             `expr:"account"`
	Amount    pkgcurrency.Amount `expr:"amount"`
	Currency  string             `expr:"currency"`
}
