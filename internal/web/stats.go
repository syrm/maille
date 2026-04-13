package web

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/syrm/maille/internal/domain/api"
)

type reporterStats interface {
	BalanceSummaryAPI(context.Context) api.BalanceSummary
	RecentTransactionsAPI(context.Context) []api.Transaction
}

type Stats struct {
	Reporter reporterStats
}

func (s Stats) Router() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", s.Get)

	return r
}

func (s Stats) Get(w http.ResponseWriter, r *http.Request) {
	balanceSummary := s.Reporter.BalanceSummaryAPI(r.Context())
	recentTransactions := s.Reporter.RecentTransactionsAPI(r.Context())

	dashboard := api.Dashboard{
		BalanceSummary:     balanceSummary,
		RecentTransactions: recentTransactions,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if err := json.NewEncoder(w).Encode(dashboard); err != nil {
		slog.Error("failed to encode response", "err", err)
	}
}
