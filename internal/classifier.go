package internal

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/expr-lang/expr"
	"github.com/samber/oops"
	"go.opentelemetry.io/otel/trace"

	"github.com/syrm/maille/internal/domain"
)

type transactionEvalCtx struct {
	Payee     string       `expr:"payee"`
	Narration string       `expr:"narration"`
	Date      time.Time    `expr:"date"`
	DayOfWeek time.Weekday `expr:"day_of_week"`
	Month     int          `expr:"month"`
	Year      int          `expr:"year"`

	Amount   float64 `expr:"amount"`
	Currency string  `expr:"currency"`
	Account  string  `expr:"account"`

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

	//accountsID := make(map[uint64]string)
	//{
	//	accounts, errAccount := c.AccountProvider.GetAll(ctx)
	//	if errAccount != nil {
	//		return oops.
	//			In("Classifier").
	//			WithContext(ctx).
	//			Wrapf(errAccount, "failed to get accounts")
	//	}
	//
	//	for _, account := range accounts {
	//		accountsID[account.ID] = string(account.Type) + ":" + account.Name
	//	}
	//}

	rules, errRule := c.RuleProvider.GetAll(ctx)

	if errRule != nil {
		return oops.
			In("Classifier").
			WithContext(ctx).
			Wrapf(errRule, "failed to get rules")
	}

	for index, r := range rules {
		fmt.Printf("Rule %#v\n", r)
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
		//accountID := tx.AccountID
		//accountName, ok := accountsID[accountID]

		//if !ok {
		//	c.Logger.Warn("accountID not found", slog.Uint64("account_id", accountID))
		//	continue
		//}

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

		fmt.Printf("posting %+v\n", posting)

		for _, rule := range rules {
			result, errExpr := expr.Run(rule.Program, posting)
			if errExpr != nil {
				fmt.Println("oups", errExpr.Error())
				continue
			}

			if result.(bool) {
				fmt.Println("ca match", posting.Payee, posting.Account)
				err := c.PostingAccountUpdater.UpdateAccount(ctx, posting.PostingIDToUpdate, rule.Account.ID)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
