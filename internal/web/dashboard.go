package web

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/CloudyKit/jet/v6"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/trace"

	"github.com/syrm/maille/internal/domain"
	"github.com/syrm/maille/internal/domain/api"
	"github.com/syrm/maille/internal/middleware"
)

type reporterDashboard interface {
	BalanceSummary(context.Context) domain.BalanceSummary
	RecentTransactions(context.Context) []api.Transaction
}

type Dashboard struct {
	Renderer Renderer
	Reporter reporterDashboard
	Tracer   trace.Tracer
	Logger   *slog.Logger
}

func (d Dashboard) Router() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", d.Get)

	return r
}

func (d Dashboard) Get(w http.ResponseWriter, r *http.Request) {
	balanceSummary := d.Reporter.BalanceSummary(r.Context())

	variables := jet.VarMap{}
	variables.Set("lang", r.Context().Value(middleware.LangKey))
	variables.Set("balanceSummary", balanceSummary)

	if err := d.Renderer.Render(w, "home", variables); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
