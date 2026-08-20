package web

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/syrm/maille/internal/domain"
)

type transactionManagerStub struct {
	updatedTransactionID uint64
	updatedAccountID     uint64
	updateErr            error
}

func (s *transactionManagerStub) List(context.Context, domain.TransactionListFilter) ([]domain.TransactionListItem, uint64, error) {
	return nil, 0, nil
}

func (s *transactionManagerStub) UpdateCategory(_ context.Context, transactionID, accountID uint64) error {
	s.updatedTransactionID = transactionID
	s.updatedAccountID = accountID
	return s.updateErr
}

func TestParseTransactionFilter(t *testing.T) {
	filter, page := parseTransactionFilter(url.Values{
		"q":        {"  amazon  "},
		"category": {"7"},
		"review":   {"uncategorized"},
		"page":     {"3"},
	})

	if filter.Search != "amazon" || filter.CategoryID != 7 || !filter.Uncategorized {
		t.Fatalf("filter = %#v", filter)
	}
	if page != 3 || filter.Limit != 25 || filter.Offset != 50 {
		t.Fatalf("page = %d, limit = %d, offset = %d", page, filter.Limit, filter.Offset)
	}
}

func TestTransactionsPostCategoryUpdatesAndPreservesFilters(t *testing.T) {
	store := &transactionManagerStub{}
	handler := Transactions{
		Transactions: store,
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	}.Router()
	form := url.Values{
		"account_id": {"9"},
		"q":          {"amazon"},
		"category":   {"7"},
		"review":     {"uncategorized"},
		"page":       {"2"},
	}
	request := httptest.NewRequest(http.MethodPost, "/42/category", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if store.updatedTransactionID != 42 || store.updatedAccountID != 9 {
		t.Fatalf("updated transaction = %d, account = %d", store.updatedTransactionID, store.updatedAccountID)
	}
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d", response.Code)
	}
	want := "/transactions?category=7&page=2&q=amazon&review=uncategorized&status=updated"
	if got := response.Header().Get("Location"); got != want {
		t.Fatalf("Location = %q, want %q", got, want)
	}
}
