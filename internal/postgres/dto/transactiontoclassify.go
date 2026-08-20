package dto

import (
	"time"

	pkgcurrency "github.com/bojanz/currency"
)

type TransactionToClassify struct {
	ID        uint64             `db:"id"`
	Date      time.Time          `db:"date"`
	Payee     string             `db:"payee"`
	Narration string             `db:"narration"`
	PostingID uint64             `db:"posting_id"`
	AccountID uint64             `db:"account_id"`
	Account   string             `db:"account"`
	Amount    pkgcurrency.Amount `db:"amount"`
	Currency  string             `db:"currency"`
}
