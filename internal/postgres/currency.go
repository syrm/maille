package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/oops"
	"go.opentelemetry.io/otel/trace"

	"github.com/syrm/maille/internal/domain"
)

type Currency struct {
	pool   *pgxpool.Pool
	tracer trace.Tracer
}

func BuildCurrency(pool *pgxpool.Pool, tracer trace.Tracer) *Currency {
	return &Currency{
		pool:   pool,
		tracer: tracer,
	}
}

func (a *Currency) GetAll(ctx context.Context) ([]domain.Currency, error) {
	ctx, span := a.tracer.Start(ctx, "Currency.GetAllWithPosting")
	defer span.End()

	rows, errQuery := a.pool.Query(
		context.WithValue(ctx, SQLName, "get all"),
		`SELECT id, name FROM currency`,
	)

	if errQuery != nil {
		return []domain.Currency{}, oops.
			In("postgres.currency").
			WithContext(ctx).
			Wrapf(errQuery, "failed to get currencies")
	}
	defer rows.Close()

	currencies, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Currency, error) {
		currency := domain.Currency{}
		errScan := row.Scan(&currency.ID, &currency.Name)

		if errScan != nil {
			return currency, oops.
				In("postgres.currency").
				WithContext(ctx).
				With("row", row).
				Wrapf(errScan, "failed to scan currency")
		}

		return currency, nil
	})

	return currencies, err
}
