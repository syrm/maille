package dto

import "time"

type RecentTransaction struct {
	Date      time.Time `db:"date"`
	Narration *string   `db:"narration"`
	Account   string    `db:"account"`
	Amount    float64   `db:"amount"`
	Currency  string    `db:"currency"`
}
