package domain

import (
	"time"

	"github.com/bojanz/currency"
)

type NetWorthHistory struct {
	Name    string
	Dates   []time.Time
	Amounts []currency.Amount
}
