package internal

import (
	"context"
	"log/slog"
	"strings"
	"time"

	pkgcurrency "github.com/bojanz/currency"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/samber/oops"
	"go.opentelemetry.io/otel/trace"

	"github.com/syrm/maille/internal/domain"
)

type transactionEvalCtx struct {
	Payee     string             `expr:"payee"`
	Narration string             `expr:"narration"`
	Date      time.Time          `expr:"date"`
	DayOfWeek int                `expr:"day_of_week"`
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

type postingClassificationUpdater interface {
	ResetClassifications(context.Context) error
	ApplyRule(context.Context, uint64, uint64, uint64) error
}

type Classifier struct {
	TransactionProvider   transactionProvider
	RuleProvider          ruleProvider
	AccountProvider       accountProvider
	PostingAccountUpdater postingClassificationUpdater
	Tracer                trace.Tracer
	Logger                *slog.Logger
}

func compileTransactionRule(rule string) (*vm.Program, error) {
	return expr.Compile(strings.TrimSpace(rule), expr.Env(transactionEvalCtx{}), expr.AsBool())
}

func (c Classifier) ValidateRule(rule string) error {
	_, err := compileTransactionRule(rule)
	return err
}

func (c Classifier) Reset(ctx context.Context) error {
	return c.PostingAccountUpdater.ResetClassifications(ctx)
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
		pgm, errComp := compileTransactionRule(r.Rule)

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
	if err := c.Reset(ctx); err != nil {
		return oops.
			In("Classifier").
			WithContext(ctx).
			Wrapf(err, "failed to reset previous classifications")
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
			Narration:         tx.Narration,
			Date:              tx.Date,
			DayOfWeek:         int(tx.Date.Weekday()),
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
				err := c.PostingAccountUpdater.ApplyRule(ctx, posting.PostingIDToUpdate, rule.ID, rule.Account.ID)
				if err != nil {
					return err
				}
				break
			}
		}
	}

	return nil
}
