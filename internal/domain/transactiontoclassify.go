package domain

import "time"

type TransactionToClassify struct {
	ID        uint64
	Date      time.Time
	Payee     string `expr:"payee"`
	PostingID uint64
	AccountID uint64
	Account   string  `expr:"account"`
	Amount    float64 `expr:"amount"`
	Currency  string  `expr:"currency"`
}
