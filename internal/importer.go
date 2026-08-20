package internal

import (
	"context"
	"errors"
	"io"
	"iter"
	"log/slog"

	"github.com/samber/oops"
	"go.opentelemetry.io/otel/trace"

	"github.com/syrm/maille/internal/domain"
	"github.com/syrm/maille/internal/ofx"
)

var defaultExpenseTransaction = domain.Account{
	Type: domain.AccountTypeExpenses,
	Name: "Other",
}

var ErrDuplicateImport = errors.New("transactions already imported")

type Parser interface {
	Parse(context.Context, io.Reader) iter.Seq2[ofx.TransactionParsed, error]
}

type TransactionStore interface {
	Save(context.Context, []domain.Transaction) error
	GetAllToClassify(context.Context, uint64, uint) (map[uint64]domain.TransactionToClassify, error)
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
	Tracer           trace.Tracer
	Logger           *slog.Logger
}

func (i Importer) Import(ctx context.Context, reader io.Reader) (int, error) {
	ctx, span := i.Tracer.Start(ctx, "Import")
	defer span.End()

	var defaultExpenseAccountID uint64
	{
		accounts, errAccount := i.AccountStore.GetAll(ctx)
		if errAccount != nil {
			return 0, oops.
				In("importer").
				WithContext(ctx).
				Wrapf(errAccount, "failed to get accounts")
		}

		for _, account := range accounts {
			if account.Type == defaultExpenseTransaction.Type &&
				account.Name == defaultExpenseTransaction.Name {
				defaultExpenseAccountID = account.ID
				break
			}
		}
	}
	if defaultExpenseAccountID == 0 {
		return 0, oops.
			In("importer").
			WithContext(ctx).
			Errorf("default expense account not found")
	}

	bankAccountsID := make(map[string]uint64)
	{
		bankAccounts, errAccount := i.BankAccountStore.GetAll(ctx)
		if errAccount != nil {
			return 0, oops.
				In("importer").
				WithContext(ctx).
				Wrapf(errAccount, "failed to get bankaccounts")
		}

		for _, bankAccount := range bankAccounts {
			bankAccountsID[bankAccount.ExternalID] = bankAccount.AccountID
		}
	}

	transactions := []domain.Transaction{}
	transactionsParsed := i.Parser.Parse(ctx, reader)

	for transactionParsed, err := range transactionsParsed {
		if err != nil {
			return 0, oops.
				In("importer").
				WithContext(ctx).
				Wrapf(err, "failed to import")
		}

		mainAccountID, ok := bankAccountsID[transactionParsed.BankAccountID]

		if !ok {
			return 0, oops.
				In("importer").
				WithContext(ctx).
				With("external_id", transactionParsed.BankAccountID).
				Errorf("bankaccount external_id not found")
		}

		amountInverse, errAmountInverse := transactionParsed.Amount.Mul("-1")

		if errAmountInverse != nil {
			return 0, oops.
				In("importer").
				With("amount", transactionParsed.Amount).
				WithContext(ctx).
				Wrapf(errAmountInverse, "failed to inverse amount")
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
				},
				{
					AccountID: defaultExpenseAccountID,
					Amount:    amountInverse,
				},
			},
		})
	}

	if len(transactions) == 0 {
		return 0, ofx.ErrNoTransactions
	}

	errSave := i.TransactionStore.Save(ctx, transactions)

	if errSave != nil {
		if errors.Is(errSave, domain.ErrDuplicateTransaction) {
			return 0, ErrDuplicateImport
		}
		return 0, oops.
			In("importer").
			WithContext(ctx).
			Wrapf(errSave, "failed to save transactions")
	}

	return len(transactions), nil
}
