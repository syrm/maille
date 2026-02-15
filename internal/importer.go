package internal

import (
	"context"
	"io"
	"iter"
	"log/slog"

	"github.com/davecgh/go-spew/spew"
	"github.com/samber/oops"
	"github.com/syrm/maille/internal/domain"
	"github.com/syrm/maille/internal/ofx"
	"go.opentelemetry.io/otel/trace"
)

var defaultExpenseTransaction = domain.Account{
	Type: domain.AccountTypeExpenses,
	Name: "Other",
}

type Parser interface {
	Parse(context.Context, io.Reader) iter.Seq2[ofx.TransactionParsed, error]
}

type TransactionStore interface {
	Save(context.Context, []domain.Transaction) error
	GetAll(context.Context, uint64, uint) ([]domain.Transaction, error)
}

type AccountStore interface {
	GetAll(context.Context) ([]domain.Account, error)
}

type BankAccountStore interface {
	GetAll(context.Context) ([]domain.BankAccount, error)
}

type CurrencyStore interface {
	GetAll(context.Context) ([]domain.Currency, error)
}

type Importer struct {
	Parser           Parser
	TransactionStore TransactionStore
	AccountStore     AccountStore
	BankAccountStore BankAccountStore
	CurrencyStore    CurrencyStore
	Tracer           trace.Tracer
	Logger           *slog.Logger
}

func (i Importer) Import(ctx context.Context, reader io.Reader) error {
	ctx, span := i.Tracer.Start(ctx, "Import")
	defer span.End()

	{
		accounts, errAccount := i.AccountStore.GetAll(ctx)
		if errAccount != nil {
			return oops.
				In("importer").
				WithContext(ctx).
				Wrapf(errAccount, "failed to get accounts")
		}

		for _, account := range accounts {
			if account.Type == defaultExpenseTransaction.Type &&
				account.Name == defaultExpenseTransaction.Name {
				defaultExpenseTransaction.ID = account.ID
				break
			}
		}
	}

	bankAccountsID := make(map[string]uint64)
	{
		bankAccounts, errAccount := i.BankAccountStore.GetAll(ctx)
		if errAccount != nil {
			return oops.
				In("importer").
				WithContext(ctx).
				Wrapf(errAccount, "failed to get bankaccounts")
		}

		for _, bankAccount := range bankAccounts {
			bankAccountsID[bankAccount.ExternalID] = bankAccount.ID
		}
	}

	currenciesID := make(map[string]domain.Currency)
	{
		currencies, errCurrency := i.CurrencyStore.GetAll(ctx)
		if errCurrency != nil {
			return oops.
				In("importer").
				WithContext(ctx).
				Wrapf(errCurrency, "failed to get currencies")
		}
		for _, currency := range currencies {
			currenciesID[currency.Name] = currency
		}
	}

	transactions := []domain.Transaction{}
	transactionsParsed := i.Parser.Parse(ctx, reader)

	for transactionParsed, err := range transactionsParsed {
		if err != nil {
			return oops.
				In("importer").
				WithContext(ctx).
				Wrapf(err, "failed to import")
		}

		mainAccountID, ok := bankAccountsID[transactionParsed.BankAccountID]

		if !ok {
			spew.Dump(transactionParsed)
			return oops.
				In("importer").
				WithContext(ctx).
				With("external_id", transactionParsed.BankAccountID).
				Errorf("bankaccount external_id not found")
		}

		currency, ok := currenciesID[transactionParsed.Currency]

		if !ok {
			return oops.
				In("importer").
				WithContext(ctx).
				With("currency", transactionParsed.Currency).
				Errorf("currency id not found")
		}

		transactions = append(transactions, domain.Transaction{
			ExternalID: transactionParsed.ID,
			Date:       transactionParsed.Date,
			Payee:      transactionParsed.Payee,
			Narration:  new(string),
			Postings: []domain.Posting{
				{
					AccountID: mainAccountID,
					Amount:    transactionParsed.Amount,
					Currency:  currency,
				},
				{
					AccountID: defaultExpenseTransaction.ID,
					Amount:    -1 * transactionParsed.Amount,
					Currency:  currency,
				},
			},
		})
	}

	// where is bulk ?
	errSave := i.TransactionStore.Save(ctx, transactions)

	if errSave != nil {
		return oops.
			In("importer").
			WithContext(ctx).
			Wrapf(errSave, "failed to save transactions")
	}

	return nil
}
