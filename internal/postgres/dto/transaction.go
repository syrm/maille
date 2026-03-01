package dto

import "time"

type Transaction struct {
	ID                    uint64    `db:"id"`
	ExternalID            string    `db:"external_id"`
	Date                  time.Time `db:"date"`
	Payee                 string    `db:"payee"`
	Narration             *string   `db:"narration"`
	Postings              []Posting `db:"posting"`
	BankAccountExternalID string    `db:"bank_account_external_id"`
}
