package domain

import "github.com/bojanz/currency"

type BreakdownCategory struct {
	Name   string
	Amount currency.Amount
}
