package web

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/syrm/maille/internal/domain"
)

type classificationRuleStoreStub struct {
	ruleID    uint64
	rule      string
	accountID uint64
	deletedID uint64
}

func (s *classificationRuleStoreStub) GetAll(context.Context) ([]domain.TransactionClassifierRule, error) {
	return nil, nil
}

func (s *classificationRuleStoreStub) Create(_ context.Context, rule string, accountID uint64) error {
	s.rule = rule
	s.accountID = accountID
	return nil
}

func (s *classificationRuleStoreStub) Update(_ context.Context, ruleID uint64, rule string, accountID uint64) error {
	s.ruleID = ruleID
	s.rule = rule
	s.accountID = accountID
	return nil
}

func (s *classificationRuleStoreStub) Delete(_ context.Context, ruleID uint64) error {
	s.deletedID = ruleID
	return nil
}

type classificationEngineStub struct {
	validateErr error
	classifies  int
	resets      int
}

func (s *classificationEngineStub) ValidateRule(string) error { return s.validateErr }
func (s *classificationEngineStub) Reset(context.Context) error {
	s.resets++
	return nil
}
func (s *classificationEngineStub) Classify(context.Context) error {
	s.classifies++
	return nil
}

func TestRulesPostUpdateValidatesPersistsAndReclassifies(t *testing.T) {
	store := &classificationRuleStoreStub{}
	engine := &classificationEngineStub{}
	handler := Rules{
		Rules: store, Engine: engine,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}.Router()
	form := url.Values{"rule": {`narration contains "SEPA"`}, "account_id": {"8"}}
	request := httptest.NewRequest(http.MethodPost, "/12", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if store.ruleID != 12 || store.rule != `narration contains "SEPA"` || store.accountID != 8 || engine.classifies != 1 {
		t.Fatalf("rule ID = %d, rule = %q, account = %d, classifies = %d", store.ruleID, store.rule, store.accountID, engine.classifies)
	}
	if got := response.Header().Get("Location"); got != "/rules?status=updated" {
		t.Fatalf("Location = %q", got)
	}
}

func TestRulesPostDeleteResetsDeletesAndReclassifies(t *testing.T) {
	store := &classificationRuleStoreStub{}
	engine := &classificationEngineStub{}
	handler := Rules{
		Rules: store, Engine: engine,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}.Router()
	request := httptest.NewRequest(http.MethodPost, "/12/delete", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if engine.resets != 1 || store.deletedID != 12 || engine.classifies != 1 {
		t.Fatalf("resets = %d, deleted rule = %d, classifies = %d", engine.resets, store.deletedID, engine.classifies)
	}
	if got := response.Header().Get("Location"); got != "/rules?status=deleted" {
		t.Fatalf("Location = %q", got)
	}
}

func TestRulesPostCreateValidatesPersistsAndRuns(t *testing.T) {
	store := &classificationRuleStoreStub{}
	engine := &classificationEngineStub{}
	handler := Rules{
		Rules: store, Engine: engine,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}.Router()
	form := url.Values{"rule": {`payee contains "AMAZON"`}, "account_id": {"7"}}
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if store.rule != `payee contains "AMAZON"` || store.accountID != 7 || engine.classifies != 1 {
		t.Fatalf("rule = %q, account = %d, classifies = %d", store.rule, store.accountID, engine.classifies)
	}
	if got := response.Header().Get("Location"); got != "/rules?status=created" {
		t.Fatalf("Location = %q", got)
	}
}

func TestRulesPostCreateRejectsInvalidExpression(t *testing.T) {
	store := &classificationRuleStoreStub{}
	engine := &classificationEngineStub{validateErr: errors.New("invalid expression")}
	handler := Rules{
		Rules: store, Engine: engine,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}.Router()
	form := url.Values{"rule": {"invalid"}, "account_id": {"7"}}
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if store.rule != "" || engine.classifies != 0 {
		t.Fatalf("invalid rule was persisted or executed")
	}
	if got := response.Header().Get("Location"); got != "/rules?error=invalid-rule" {
		t.Fatalf("Location = %q", got)
	}
}
