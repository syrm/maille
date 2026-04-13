package dto

import (
	"time"

	pkgcurrency "github.com/bojanz/currency"
)

type RecentTransaction struct {
	Date    time.Time          `db:"date"`
	Payee   *string            `db:"payee"`
	Account string             `db:"account"`
	Amount  pkgcurrency.Amount `db:"amount"`
	Alias   string             `db:"alias"`
	Icon    string             `db:"icon"`
	Color   string             `db:"color"`
}
