package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/oops"
	"github.com/syrm/maille/internal"
	"go.opentelemetry.io/otel/trace"
)

type Account struct {
	pool   *pgxpool.Pool
	tracer trace.Tracer
}

func BuildAccount(ctx context.Context, pool *pgxpool.Pool, tracer trace.Tracer) (*Transaction, error) {
	return &Transaction{
		pool:   pool,
		tracer: tracer,
	}, nil
}

func (s *Transaction) GetAll(ctx context.Context) ([]internal.Account, error) {
	ctx, span := s.tracer.Start(ctx, "GetAll")
	defer span.End()

	rows, errQuery := s.pool.Query(
		context.WithValue(ctx, SQLName, "get all"),
		`SELECT id, type, name
		FROM account
		`,
	)

	if errQuery != nil {
		return []internal.Account{}, oops.
			In("Store").
			WithContext(ctx).
			Wrapf(errQuery, "failed to get accounts")
	}
	defer rows.Close()

	var accounts []internal.Account

	for rows.Next() {
		var id int
		var accountType internal.AccountType
		var name string

		rows.Scan(&id, &accountType, &name)

		account := internal.Account{}
		account.ID = uint64(id)
		account.Type = accountType
		account.Name = name

		accounts = append(accounts, account)
	}

	if err := rows.Err(); err != nil {
		return accounts, fmt.Errorf("rows iteration: %w", err)
	}

	return accounts, nil
}
