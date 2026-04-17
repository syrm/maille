package dto

import (
	"time"
)

type NetWorthHistory struct {
	Name    string      `db:"name"`
	Dates   []time.Time `db:"dates"`
	Amounts []float64   `db:"amounts"`
}
