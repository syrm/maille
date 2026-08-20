package internal

import (
	"context"
	"log/slog"
	"time"

	pkgcurrency "github.com/bojanz/currency"
	"github.com/expr-lang/expr"
	"github.com/samber/oops"
	"go.opentelemetry.io/otel/trace"

	"github.com/syrm/maille/internal/domain"
)

type transactionEvalCtx struct {
	Payee     string             `expr:"payee"`
	Narration string             `expr:"narration"`
	Date      time.Time          `expr:"date"`
	DayOfWeek time.Weekday       `expr:"day_of_week"`
	Month     int                `expr:"month"`
	Year      int                `expr:"year"`
	Amount    pkgcurrency.Amount `expr:"amount"`
	Currency  string             `expr:"currency"`
	Account   string             `expr:"account"`

	PostingIDToUpdate uint64
}

type accountProvider interface {
	GetAll(context.Context) ([]domain.Account, error)
}

type transactionProvider interface {
	GetAllToClassify(context.Context, uint64, uint) (map[uint64]domain.TransactionToClassify, error)
}

type ruleProvider interface {
	GetAll(context.Context) ([]domain.TransactionClassifierRule, error)
}

type postingAccountUpdater interface {
	UpdateAccount(context.Context, uint64, uint64) error
}

type Classifier struct {
	TransactionProvider   transactionProvider
	RuleProvider          ruleProvider
	AccountProvider       accountProvider
	PostingAccountUpdater postingAccountUpdater
	Tracer                trace.Tracer
	Logger                *slog.Logger
}

func (c Classifier) Classify(ctx context.Context) error {
	ctx, span := c.Tracer.Start(ctx, "Classifier")
	defer span.End()

	rules, errRule := c.RuleProvider.GetAll(ctx)

	if errRule != nil {
		return oops.
			In("Classifier").
			WithContext(ctx).
			Wrapf(errRule, "failed to get rules")
	}
	for index, r := range rules {
		pgm, errComp := expr.Compile(r.Rule, expr.Env(transactionEvalCtx{}), expr.AsBool())

		if errComp != nil {
			return oops.
				In("Classifier").
				WithContext(ctx).
				With("rule", r.Rule).
				Wrapf(errComp, "failed to compile rule")
		}

		r.Program = pgm
		rules[index] = r
	}

	txs, errTx := c.TransactionProvider.GetAllToClassify(ctx, 0, 100_000)

	if errTx != nil {
		return oops.
			In("Classifier").
			WithContext(ctx).
			Wrapf(errTx, "failed to get transactions")
	}

	for _, tx := range txs {
		posting := transactionEvalCtx{
			Payee:             tx.Payee,
			Date:              tx.Date,
			DayOfWeek:         tx.Date.Weekday(),
			Month:             int(tx.Date.Month()),
			Year:              tx.Date.Year(),
			Amount:            tx.Amount,
			Currency:          tx.Currency,
			Account:           tx.Account,
			PostingIDToUpdate: tx.PostingID,
		}

		for _, rule := range rules {
			result, errExpr := expr.Run(rule.Program, posting)
			if errExpr != nil {
				c.Logger.WarnContext(ctx, "failed to evaluate classification rule", slog.Any("error", errExpr))
				continue
			}

			if result.(bool) {
				err := c.PostingAccountUpdater.UpdateAccount(ctx, posting.PostingIDToUpdate, rule.Account.ID)
				if err != nil {
					return err
				}
				break
			}
		}
	}

	return nil
}
