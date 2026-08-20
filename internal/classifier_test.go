package internal

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/bojanz/currency"
	"go.opentelemetry.io/otel"

	"github.com/syrm/maille/internal/domain"
)

type classificationTransactionsStub map[uint64]domain.TransactionToClassify

func (s classificationTransactionsStub) GetAllToClassify(context.Context, uint64, uint) (map[uint64]domain.TransactionToClassify, error) {
	return s, nil
}

type classificationRulesStub []domain.TransactionClassifierRule

func (s classificationRulesStub) GetAll(context.Context) ([]domain.TransactionClassifierRule, error) {
	return s, nil
}

type postingUpdaterStub struct {
	postingID uint64
	ruleID    uint64
	accountID uint64
	resets    int
}

func (s *postingUpdaterStub) ResetClassifications(context.Context) error {
	s.resets++
	return nil
}

func (s *postingUpdaterStub) ApplyRule(_ context.Context, postingID, ruleID, accountID uint64) error {
	s.postingID = postingID
	s.ruleID = ruleID
	s.accountID = accountID
	return nil
}

func TestClassifierValidatesFullExpressionContext(t *testing.T) {
	classifier := Classifier{}
	validRules := []string{
		`payee contains "AMAZON"`,
		`narration contains "SEPA" and currency == "EUR"`,
		`month == 8 and year == 2026 and day_of_week == 4`,
		`amount.IsNegative() and account == "Other"`,
	}
	for _, rule := range validRules {
		if err := classifier.ValidateRule(rule); err != nil {
			t.Errorf("ValidateRule(%q) error = %v", rule, err)
		}
	}
	if err := classifier.ValidateRule(`payee ??? "AMAZON"`); err == nil {
		t.Fatal("ValidateRule() accepted invalid expression")
	}
}

func TestClassifierUsesNarrationAndCurrency(t *testing.T) {
	amount, err := currency.NewAmount("-42.50", "EUR")
	if err != nil {
		t.Fatal(err)
	}
	updater := &postingUpdaterStub{}
	classifier := Classifier{
		TransactionProvider: classificationTransactionsStub{1: {
			ID: 1, Date: time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC), Payee: "MARKET",
			Narration: "PAIEMENT SEPA", PostingID: 99, Account: "Other", Amount: amount, Currency: "EUR",
		}},
		RuleProvider: classificationRulesStub{{
			ID:      3,
			Rule:    `narration contains "SEPA" and currency == "EUR"`,
			Account: domain.Account{ID: 7},
		}},
		PostingAccountUpdater: updater,
		Tracer:                otel.Tracer("test"),
		Logger:                slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	if err := classifier.Classify(context.Background()); err != nil {
		t.Fatalf("Classify() error = %v", err)
	}
	if updater.resets != 1 || updater.postingID != 99 || updater.ruleID != 3 || updater.accountID != 7 {
		t.Fatalf("resets = %d, updated posting = %d, rule = %d, account = %d", updater.resets, updater.postingID, updater.ruleID, updater.accountID)
	}
}
