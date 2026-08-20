package web

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/CloudyKit/jet/v6"
	"github.com/go-chi/chi/v5"

	"github.com/syrm/maille/internal/domain"
	"github.com/syrm/maille/internal/middleware"
)

const transactionsPerPage = 25

type transactionManager interface {
	List(context.Context, domain.TransactionListFilter) ([]domain.TransactionListItem, uint64, error)
	UpdateCategory(context.Context, uint64, uint64) error
}

type categoryProvider interface {
	GetAll(context.Context) ([]domain.Account, error)
}

type Transactions struct {
	Renderer     Renderer
	Transactions transactionManager
	Accounts     categoryProvider
	Logger       *slog.Logger
}

type transactionPagination struct {
	Start       uint64
	End         uint64
	Total       uint64
	CurrentPage uint64
	HasPrevious bool
	PreviousURL string
	HasNext     bool
	NextURL     string
}

type transactionCategory struct {
	ID    uint64
	Label string
	Icon  string
}

func (t Transactions) Router() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", t.Get)
	r.Post("/{transactionID}/category", t.PostCategory)
	return r
}

func (t Transactions) Get(w http.ResponseWriter, r *http.Request) {
	filter, page := parseTransactionFilter(r.URL.Query())
	transactions, total, err := t.Transactions.List(r.Context(), filter)
	if err != nil {
		t.Logger.ErrorContext(r.Context(), "failed to list transactions", slog.Any("error", err))
		http.Error(w, "Impossible de charger les transactions.", http.StatusInternalServerError)
		return
	}
	if page > 1 && len(transactions) == 0 {
		http.Redirect(w, r, transactionListURL(filter, 1, "", ""), http.StatusSeeOther)
		return
	}

	accounts, err := t.Accounts.GetAll(r.Context())
	if err != nil {
		t.Logger.ErrorContext(r.Context(), "failed to list transaction categories", slog.Any("error", err))
		http.Error(w, "Impossible de charger les catégories.", http.StatusInternalServerError)
		return
	}
	categoryAccounts := make([]domain.Account, 0, len(accounts))
	aliases := make(map[string]int)
	for _, account := range accounts {
		if account.Type == domain.AccountTypeExpenses || account.Type == domain.AccountTypeIncome {
			categoryAccounts = append(categoryAccounts, account)
			aliases[account.Alias]++
		}
	}
	categories := make([]transactionCategory, 0, len(categoryAccounts))
	for _, account := range categoryAccounts {
		label := account.Alias
		if aliases[account.Alias] > 1 {
			nameParts := strings.Split(account.Name, ":")
			label += " · " + nameParts[len(nameParts)-1]
		}
		categories = append(categories, transactionCategory{ID: account.ID, Label: label, Icon: account.Icon})
	}

	pagination := buildTransactionPagination(filter, page, total, uint64(len(transactions)))
	variables := jet.VarMap{}
	variables.Set("lang", r.Context().Value(middleware.LangKey))
	variables.Set("pageTitle", "Transactions")
	variables.Set("currentPage", "transactions")
	variables.Set("transactions", transactions)
	variables.Set("hasTransactions", len(transactions) > 0)
	variables.Set("categories", categories)
	variables.Set("filter", filter)
	variables.Set("page", page)
	variables.Set("pagination", pagination)
	variables.Set("notice", transactionNotice(r.URL.Query().Get("status")))
	variables.Set("errorMessage", transactionError(r.URL.Query().Get("error")))

	if err := t.Renderer.Render(w, "transactions", variables); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (t Transactions) PostCategory(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		redirectTransactionResult(w, r, "", "invalid-category")
		return
	}
	transactionID, errTransactionID := strconv.ParseUint(chi.URLParam(r, "transactionID"), 10, 64)
	accountID, errAccountID := strconv.ParseUint(r.FormValue("account_id"), 10, 64)
	if errTransactionID != nil || transactionID == 0 || errAccountID != nil || accountID == 0 {
		redirectTransactionResult(w, r, "", "invalid-category")
		return
	}

	if err := t.Transactions.UpdateCategory(r.Context(), transactionID, accountID); err != nil {
		t.Logger.ErrorContext(r.Context(), "failed to update transaction category", slog.Any("error", err))
		redirectTransactionResult(w, r, "", "update-failed")
		return
	}
	redirectTransactionResult(w, r, "updated", "")
}

func parseTransactionFilter(values url.Values) (domain.TransactionListFilter, uint64) {
	search := strings.TrimSpace(values.Get("q"))
	searchRunes := []rune(search)
	if len(searchRunes) > 100 {
		search = string(searchRunes[:100])
	}
	categoryID, _ := strconv.ParseUint(values.Get("category"), 10, 64)
	page, _ := strconv.ParseUint(values.Get("page"), 10, 64)
	if page < 1 {
		page = 1
	} else if page > 1_000_000 {
		page = 1_000_000
	}

	return domain.TransactionListFilter{
		Search:        search,
		CategoryID:    categoryID,
		Uncategorized: values.Get("review") == "uncategorized",
		Limit:         transactionsPerPage,
		Offset:        uint((page - 1) * transactionsPerPage),
	}, page
}

func buildTransactionPagination(filter domain.TransactionListFilter, page, total, count uint64) transactionPagination {
	offset := uint64(filter.Offset)
	pagination := transactionPagination{Total: total, CurrentPage: page}
	if count > 0 {
		pagination.Start = offset + 1
		pagination.End = offset + count
	}
	if page > 1 {
		pagination.HasPrevious = true
		pagination.PreviousURL = transactionListURL(filter, page-1, "", "")
	}
	if offset+count < total {
		pagination.HasNext = true
		pagination.NextURL = transactionListURL(filter, page+1, "", "")
	}
	return pagination
}

func redirectTransactionResult(w http.ResponseWriter, r *http.Request, status, errorCode string) {
	filter, page := parseTransactionFilter(r.PostForm)
	http.Redirect(w, r, transactionListURL(filter, page, status, errorCode), http.StatusSeeOther)
}

func transactionListURL(filter domain.TransactionListFilter, page uint64, status, errorCode string) string {
	query := url.Values{}
	if filter.Search != "" {
		query.Set("q", filter.Search)
	}
	if filter.CategoryID != 0 {
		query.Set("category", strconv.FormatUint(filter.CategoryID, 10))
	}
	if filter.Uncategorized {
		query.Set("review", "uncategorized")
	}
	if page > 1 {
		query.Set("page", strconv.FormatUint(page, 10))
	}
	if status != "" {
		query.Set("status", status)
	}
	if errorCode != "" {
		query.Set("error", errorCode)
	}
	if len(query) == 0 {
		return "/transactions"
	}
	return "/transactions?" + query.Encode()
}

func transactionNotice(status string) string {
	if status == "updated" {
		return "Catégorie mise à jour."
	}
	return ""
}

func transactionError(code string) string {
	switch code {
	case "invalid-category":
		return "La catégorie sélectionnée est invalide."
	case "update-failed":
		return "La catégorie n’a pas pu être mise à jour."
	default:
		return ""
	}
}
