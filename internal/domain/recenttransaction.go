package domain

import (
	"time"

	pkgcurrency "github.com/bojanz/currency"
)

type RecentTransaction struct {
	Date    time.Time
	Payee   *string
	Account string
	Amount  pkgcurrency.Amount
	Alias   string
	Icon    string
	Color   string
}
