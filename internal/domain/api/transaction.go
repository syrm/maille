package api

import "time"

type Transaction struct {
	Date      time.Time
	Narration *string
	Account   string
	Amount    float64
	Currency  string
}
