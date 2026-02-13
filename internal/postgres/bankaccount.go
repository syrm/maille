package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/oops"
	"github.com/syrm/maille/internal/domain"
	"go.opentelemetry.io/otel/trace"
)

type BankAccount struct {
	pool   *pgxpool.Pool
	tracer trace.Tracer
}

func BuildBankAccount(ctx context.Context, pool *pgxpool.Pool, tracer trace.Tracer) *BankAccount {
	return &BankAccount{
		pool:   pool,
		tracer: tracer,
	}
}

func (a *BankAccount) GetAll(ctx context.Context) ([]domain.BankAccount, error) {
	ctx, span := a.tracer.Start(ctx, "GetAll")
	defer span.End()

	rows, errQuery := a.pool.Query(
		context.WithValue(ctx, SQLName, "get all"),
		`SELECT id, name, account_id, external_id FROM bank_account`,
	)

	if errQuery != nil {
		return []domain.BankAccount{}, oops.
			In("postgres.bankaccount").
			WithContext(ctx).
			Wrapf(errQuery, "failed to get bankaccounts")
	}
	defer rows.Close()

	bankAccounts, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.BankAccount, error) {
		bankAccount := domain.BankAccount{}
		errScan := row.Scan(&bankAccount.ID, &bankAccount.Name, &bankAccount.AccountID, &bankAccount.ExternalID)

		if errScan != nil {
			return bankAccount, oops.
				In("postgres.bankaccount").
				WithContext(ctx).
				With("row", row).
				Wrapf(errScan, "failed to scan bankaccount")
		}

		return bankAccount, nil
	})

	return bankAccounts, err
}
