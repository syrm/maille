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
		`SELECT tcr.id, tcr.rule, tcr.account_id AS account_id, account.type AS account_type,
		        account.name AS account_name, account.alias AS account_alias, account.icon AS account_icon,
		        account.color AS account_color
		FROM transaction_classifier_rule tcr
		INNER JOIN account ON (account.id = tcr.account_id)
		ORDER BY tcr.id
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
			errScan := row.Scan(
				&tcr.ID,
				&tcr.Rule,
				&account.ID,
				&account.Type,
				&account.Name,
				&account.Alias,
				&account.Icon,
				&account.Color,
			)

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

func (t *TransactionClassifierRule) Create(ctx context.Context, rule string, accountID uint64) error {
	ctx, span := t.tracer.Start(ctx, "TransactionClassifierRule.Create")
	defer span.End()

	result, errExec := t.pool.Exec(
		context.WithValue(ctx, SQLName, "create classification rule"),
		`INSERT INTO transaction_classifier_rule (rule, account_id)
		 SELECT @rule, id FROM account
		 WHERE id = @accountID AND type IN ('Expenses', 'Income')`,
		pgx.NamedArgs{"rule": rule, "accountID": accountID},
	)
	if errExec != nil {
		return oops.In("postgres.transactionclassifierrule").WithContext(ctx).Wrapf(errExec, "failed to create rule")
	}
	if result.RowsAffected() != 1 {
		return oops.In("postgres.transactionclassifierrule").WithContext(ctx).With("account_id", accountID).Errorf("category not found")
	}
	return nil
}

func (t *TransactionClassifierRule) Update(ctx context.Context, id uint64, rule string, accountID uint64) error {
	ctx, span := t.tracer.Start(ctx, "TransactionClassifierRule.Update")
	defer span.End()

	result, errExec := t.pool.Exec(
		context.WithValue(ctx, SQLName, "update classification rule"),
		`UPDATE transaction_classifier_rule
		 SET rule = @rule, account_id = @accountID
		 WHERE id = @id
		   AND EXISTS (
		       SELECT 1 FROM account
		       WHERE account.id = @accountID AND account.type IN ('Expenses', 'Income')
		   )`,
		pgx.NamedArgs{"id": id, "rule": rule, "accountID": accountID},
	)
	if errExec != nil {
		return oops.In("postgres.transactionclassifierrule").WithContext(ctx).Wrapf(errExec, "failed to update rule")
	}
	if result.RowsAffected() != 1 {
		return oops.In("postgres.transactionclassifierrule").WithContext(ctx).With("rule_id", id).With("account_id", accountID).Errorf("rule or category not found")
	}
	return nil
}

func (t *TransactionClassifierRule) Delete(ctx context.Context, id uint64) error {
	ctx, span := t.tracer.Start(ctx, "TransactionClassifierRule.Delete")
	defer span.End()

	result, errExec := t.pool.Exec(
		context.WithValue(ctx, SQLName, "delete classification rule"),
		`DELETE FROM transaction_classifier_rule WHERE id = @id`,
		pgx.NamedArgs{"id": id},
	)
	if errExec != nil {
		return oops.In("postgres.transactionclassifierrule").WithContext(ctx).Wrapf(errExec, "failed to delete rule")
	}
	if result.RowsAffected() != 1 {
		return oops.In("postgres.transactionclassifierrule").WithContext(ctx).With("rule_id", id).Errorf("rule not found")
	}
	return nil
}
