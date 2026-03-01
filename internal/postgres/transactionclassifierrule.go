package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/oops"
	"go.opentelemetry.io/otel/trace"

	"github.com/syrm/maille/internal/domain"
)

type TransactionClassifierRule struct {
	pool   *pgxpool.Pool
	tracer trace.Tracer
}

func BuildTransactionClassifierRule(ctx context.Context, pool *pgxpool.Pool, tracer trace.Tracer) *TransactionClassifierRule {
	return &TransactionClassifierRule{
		pool:   pool,
		tracer: tracer,
	}
}

func (t *TransactionClassifierRule) GetAll(ctx context.Context) ([]domain.TransactionClassifierRule, error) {
	ctx, span := t.tracer.Start(ctx, "TransactionClassifierRule.GetAll")
	defer span.End()

	rows, errQuery := t.pool.Query(
		context.WithValue(ctx, SQLName, "get all"),
		`SELECT tcr.id, tcr.rule, tcr.account_id AS account_id, account.type AS account_type, account.name AS account_name
		FROM transaction_classifier_rule tcr
		INNER JOIN account ON (account.id = tcr.account_id)
		`,
	)

	if errQuery != nil {
		return []domain.TransactionClassifierRule{}, oops.
			In("postgres.transactionclassifierrule").
			WithContext(ctx).
			Wrapf(errQuery, "failed to get transactionClassifierRules")
	}
	defer rows.Close()

	transactionClassifierRules, err := pgx.CollectRows(
		rows,
		func(row pgx.CollectableRow) (domain.TransactionClassifierRule, error) {
			tcr := domain.TransactionClassifierRule{}
			account := domain.Account{}
			errScan := row.Scan(&tcr.ID, &tcr.Rule, &account.ID, &account.Type, &account.Name)

			if errScan != nil {
				return domain.TransactionClassifierRule{}, oops.
					In("postgres.transactionclassifierrule").
					WithContext(ctx).
					With("row", row).
					Wrapf(errScan, "failed to scan transactionClassifierRules")
			}

			if !account.Type.IsValid() {
				return domain.TransactionClassifierRule{}, oops.
					In("postgres.transactionclassifierrule").
					WithContext(ctx).
					With("account_type", account.Type).
					Errorf("account type is invalid")
			}

			tcr.Account = account

			return tcr, nil
		})

	return transactionClassifierRules, err
}
