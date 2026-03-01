package dto

type Posting struct {
	ID        uint64   `db:"id"`
	AccountID uint64   `db:"account_id"`
	Amount    float64  `db:"amount"`
	Currency  Currency `db:"currency"`
}
