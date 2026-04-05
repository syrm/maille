package domain

import "time"

type Transaction struct {
	ID                    uint64
	ExternalID            string
	Date                  time.Time
	Payee                 string `expr:"payee"`
	Narration             *string
	Postings              []Posting `expr:"postings"`
	BankAccountExternalID string
}
