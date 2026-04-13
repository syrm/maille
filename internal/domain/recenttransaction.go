package domain

import (
	"time"

	pkgcurrency "github.com/bojanz/currency"
)

type RecentTransaction struct {
	Date      time.Time
	Narration *string
	Account   string
	Amount    pkgcurrency.Amount
}
