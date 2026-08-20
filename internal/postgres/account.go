package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/oops"
	"go.opentelemetry.io/otel/trace"

	"github.com/syrm/maille/internal/domain"
)

type Account struct {
	pool   *pgxpool.Pool
	tracer trace.Tracer
}

func BuildAccount(pool *pgxpool.Pool, tracer trace.Tracer) *Account {
	return &Account{
		pool:   pool,
		tracer: tracer,
	}
}

func (a *Account) GetAll(ctx context.Context) ([]domain.Account, error) {
	ctx, span := a.tracer.Start(ctx, "Account.GetAllWithPosting")
	defer span.End()

	rows, errQuery := a.pool.Query(
		context.WithValue(ctx, SQLName, "get all"),
		`SELECT id, type, name, alias, icon, color FROM account ORDER BY type, name`,
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
		errScan := row.Scan(&account.ID, &account.Type, &account.Name, &account.Alias, &account.Icon, &account.Color)

		if errScan != nil {
			return account, oops.
				In("postgres.account").
				WithContext(ctx).
				With("row", row).
				Wrapf(errScan, "failed to scan account")
		}

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
