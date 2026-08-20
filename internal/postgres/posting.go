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

func (s *Posting) ApplyRule(ctx context.Context, postingID, ruleID, accountID uint64) error {
	ctx, span := s.tracer.Start(ctx, "Posting.ApplyRule")
	defer span.End()

	_, errExec := s.pool.Exec(
		context.WithValue(ctx, SQLName, "update account"),
		`WITH updated_posting AS (
		    UPDATE posting
		    SET account_id = @accountID, tcr_id = @ruleID
		    WHERE id = @id
		    RETURNING transaction_id
		)
		UPDATE transaction
		SET classified_at = CURRENT_TIMESTAMP
		WHERE id = (SELECT transaction_id FROM updated_posting)`,
		pgx.NamedArgs{
			"accountID": accountID,
			"ruleID":    ruleID,
			"id":        postingID,
		},
	)

	return oops.
		In("postgres.posting").
		WithContext(ctx).
		With("posting_id", postingID).
		With("rule_id", ruleID).
		With("account_id", accountID).
		Wrapf(errExec, "failed to update posting")
}

func (s *Posting) ResetClassifications(ctx context.Context) error {
	ctx, span := s.tracer.Start(ctx, "Posting.ResetClassifications")
	defer span.End()

	_, errExec := s.pool.Exec(
		context.WithValue(ctx, SQLName, "reset rule classifications"),
		`WITH default_account AS (
		    SELECT id FROM account WHERE type = 'Expenses' AND name = 'Other' LIMIT 1
		), reset_postings AS (
		    UPDATE posting
		    SET account_id = default_account.id, tcr_id = NULL
		    FROM default_account, account AS current_account, transaction
		    WHERE posting.account_id = current_account.id
		      AND posting.transaction_id = transaction.id
		      AND current_account.type IN ('Expenses', 'Income')
		      AND (posting.tcr_id IS NOT NULL OR transaction.classified_at IS NOT NULL)
		    RETURNING posting.transaction_id
		)
		UPDATE transaction
		SET classified_at = NULL
		WHERE id IN (SELECT transaction_id FROM reset_postings)`,
	)

	return oops.
		In("postgres.posting").
		WithContext(ctx).
		Wrapf(errExec, "failed to reset rule classifications")
}
