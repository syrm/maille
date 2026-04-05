package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/oops"
	"go.opentelemetry.io/otel/trace"
)

type Posting struct {
	pool   *pgxpool.Pool
	tracer trace.Tracer
}

func BuildPosting(pool *pgxpool.Pool, tracer trace.Tracer) *Posting {
	return &Posting{
		pool:   pool,
		tracer: tracer,
	}
}

func (s *Posting) UpdateAccount(ctx context.Context, postingID uint64, accountID uint64) error {
	ctx, span := s.tracer.Start(ctx, "Posting.update")
	defer span.End()

	_, errExec := s.pool.Exec(
		context.WithValue(ctx, SQLName, "update account"),
		`UPDATE posting SET account_id = @accountID WHERE id = @id`,
		pgx.NamedArgs{
			"accountID": accountID,
			"id":        postingID,
		},
	)

	return oops.
		In("postgres.posting").
		WithContext(ctx).
		With("posting_id", postingID).
		With("account_id", accountID).
		Wrapf(errExec, "failed to update posting")
}
