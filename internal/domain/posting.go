package domain

import pkgcurrency "github.com/bojanz/currency"

type Posting struct {
	ID        uint64
	AccountID uint64
	Amount    pkgcurrency.Amount `expr:"amount"`
}
