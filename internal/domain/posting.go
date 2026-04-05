package domain

type Posting struct {
	ID        uint64
	AccountID uint64
	Amount    float64 `expr:"amount"`
	Currency  Currency
}
