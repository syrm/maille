package internal

import (
	"context"
	"errors"
	"io"
	"iter"
	"strings"
	"testing"
	"time"

	"github.com/bojanz/currency"
	"go.opentelemetry.io/otel"

	"github.com/syrm/maille/internal/domain"
	"github.com/syrm/maille/internal/ofx"
)

type parserStub struct {
	transactions []ofx.TransactionParsed
}

func (p parserStub) Parse(context.Context, io.Reader) iter.Seq2[ofx.TransactionParsed, error] {
	return func(yield func(ofx.TransactionParsed, error) bool) {
		for _, transaction := range p.transactions {
			if !yield(transaction, nil) {
				return
			}
		}
	}
}

type accountStoreStub []domain.Account

func (s accountStoreStub) GetAll(context.Context) ([]domain.Account, error) { return s, nil }

type bankAccountStoreStub []domain.BankAccount

func (s bankAccountStoreStub) GetAll(context.Context) ([]domain.BankAccount, error) { return s, nil }

type transactionStoreStub struct {
	saved   []domain.Transaction
	saveErr error
}

func (s *transactionStoreStub) Save(_ context.Context, transactions []domain.Transaction) error {
	s.saved = transactions
	return s.saveErr
}

func (s *transactionStoreStub) GetAllToClassify(context.Context, uint64, uint) (map[uint64]domain.TransactionToClassify, error) {
	return nil, nil
}

func TestImporterBuildsBalancedTransactionsWithAccountID(t *testing.T) {
	amount, err := currency.NewAmount("-42.50", "EUR")
	if err != nil {
		t.Fatal(err)
	}
	store := &transactionStoreStub{}
	importer := Importer{
		Parser: parserStub{transactions: []ofx.TransactionParsed{{
			ID: "fitid-1", Date: time.Date(2026, 8, 19, 0, 0, 0, 0, time.UTC), Payee: "AMAZON",
			Narration: "ACHAT CARTE", Amount: amount, BankAccountID: "bank-external-id",
		}}},
		TransactionStore: store,
		AccountStore:     accountStoreStub{{ID: 7, Type: domain.AccountTypeExpenses, Name: "Other"}},
		BankAccountStore: bankAccountStoreStub{{ID: 9, AccountID: 42, ExternalID: "bank-external-id"}},
		Tracer:           otel.Tracer("test"),
	}

	count, err := importer.Import(context.Background(), strings.NewReader("OFX"))
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if count != 1 || len(store.saved) != 1 {
		t.Fatalf("Import() count = %d, saved = %d", count, len(store.saved))
	}
	if store.saved[0].Narration == nil || *store.saved[0].Narration != "ACHAT CARTE" {
		t.Errorf("saved narration = %v", store.saved[0].Narration)
	}
	postings := store.saved[0].Postings
	if postings[0].AccountID != 42 {
		t.Errorf("bank posting account = %d, want accounting account 42", postings[0].AccountID)
	}
	if postings[1].AccountID != 7 || postings[1].Amount.Number() != "42.50" {
		t.Errorf("balancing posting = %#v", postings[1])
	}
}

func TestImporterReportsDuplicateImport(t *testing.T) {
	amount, _ := currency.NewAmount("1", "EUR")
	store := &transactionStoreStub{saveErr: domain.ErrDuplicateTransaction}
	importer := Importer{
		Parser:           parserStub{transactions: []ofx.TransactionParsed{{ID: "fitid-1", Date: time.Now(), Payee: "Payee", Amount: amount, BankAccountID: "bank"}}},
		TransactionStore: store,
		AccountStore:     accountStoreStub{{ID: 7, Type: domain.AccountTypeExpenses, Name: "Other"}},
		BankAccountStore: bankAccountStoreStub{{AccountID: 42, ExternalID: "bank"}},
		Tracer:           otel.Tracer("test"),
	}

	_, err := importer.Import(context.Background(), strings.NewReader("OFX"))
	if !errors.Is(err, ErrDuplicateImport) {
		t.Fatalf("Import() error = %v, want ErrDuplicateImport", err)
	}
}
