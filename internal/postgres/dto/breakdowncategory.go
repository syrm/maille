package dto

type BreakdownCategory struct {
	Name   string  `db:"name"`
	Amount float64 `db:"amount"`
}
