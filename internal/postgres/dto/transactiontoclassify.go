package dto

import "time"

type TransactionToClassify struct {
	ID        uint64    `db:"id"`
	Date      time.Time `db:"date"`
	Payee     string    `db:"payee"`
	PostingID uint64    `db:"posting_id"`
	AccountID uint64    `db:"account_id"`
	Account   string    `expr:"account"`
	Amount    float64   `expr:"amount"`
	Currency  string    `expr:"currency"`
}
