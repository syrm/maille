package dto

import (
	"time"

	pkgcurrency "github.com/bojanz/currency"
)

type RecentTransaction struct {
	Date      time.Time          `db:"date"`
	Narration *string            `db:"narration"`
	Account   string             `db:"account"`
	Amount    pkgcurrency.Amount `db:"amount"`
}
