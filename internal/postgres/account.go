package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/oops"
	"github.com/syrm/maille/internal/domain"
	"go.opentelemetry.io/otel/trace"
)

type Account struct {
	pool   *pgxpool.Pool
	tracer trace.Tracer
}

func BuildAccount(ctx context.Context, pool *pgxpool.Pool, tracer trace.Tracer) *Account {
	return &Account{
		pool:   pool,
		tracer: tracer,
	}
}

func (a *Account) GetAll(ctx context.Context) ([]domain.Account, error) {
	ctx, span := a.tracer.Start(ctx, "GetAll")
	defer span.End()

	rows, errQuery := a.pool.Query(
		context.WithValue(ctx, SQLName, "get all"),
		`SELECT id, type, name FROM account`,
	)

	if errQuery != nil {
		return []domain.Account{}, oops.
			In("postgres.account").
			WithContext(ctx).
			Wrapf(errQuery, "failed to get accounts")
	}
	defer rows.Close()

	accounts, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Account, error) {
		account := domain.Account{}
		errScan := row.Scan(&account.ID, &account.Type, &account.Name)

		if errScan != nil {
			return account, oops.
				In("postgres.account").
				WithContext(ctx).
				With("row", row).
				Wrapf(errScan, "failed to scan account")
		}

		account.Type = domain.AccountType(account.Type)

		if !account.Type.IsValid() {
			return account, oops.
				In("postgres.account").
				WithContext(ctx).
				With("type", account.Type).
				Errorf("account type is invalid")
		}

		return account, nil
	})

	return accounts, err
}
