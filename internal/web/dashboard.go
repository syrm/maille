package web

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/CloudyKit/jet/v6"
	"github.com/go-chi/chi/v5"
	"github.com/goodsign/monday"
	"go.opentelemetry.io/otel/trace"

	"github.com/syrm/maille/internal/domain"
	"github.com/syrm/maille/internal/middleware"
)

type reporterDashboard interface {
	BalanceSummary(context.Context) domain.BalanceSummary
	RecentTransactions(context.Context) []domain.RecentTransaction
	NetWorthHistory(ctx context.Context) []domain.NetWorthHistory
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

type Series struct {
	Name   string    `json:"name"`
	Data   []float64 `json:"data"`
	Color  *string   `json:"color,omitempty"`
	Area   bool      `json:"area"`
	Dashed bool      `json:"dashed"`
}

type Options struct {
	Width  *float64 `json:"width,omitempty"`
	Height *float64 `json:"height,omitempty"`
	Labels []string `json:"labels,omitempty"`
	YTicks *int     `json:"yTicks,omitempty"`
	Smooth bool     `json:"smooth"`
	Points bool     `json:"points"`
	Colors []string `json:"colors,omitempty"`
}

type Chart struct {
	Series  []Series `json:"series"`
	Options *Options `json:"options,omitempty"`
}

func (d Dashboard) Get(w http.ResponseWriter, r *http.Request) {
	balanceSummary := d.Reporter.BalanceSummary(r.Context())
	recentTransactions := d.Reporter.RecentTransactions(r.Context())
	netWorthHistoryData := d.Reporter.NetWorthHistory(r.Context())

	variables := jet.VarMap{}
	variables.Set("lang", r.Context().Value(middleware.LangKey))
	variables.Set("balanceSummary", balanceSummary)
	variables.Set("recentTransactions", recentTransactions)
	variables.Set("netWorthHistory", buildChart(r.Context().Value(middleware.LangKey).(string), netWorthHistoryData))

	if err := d.Renderer.Render(w, "home", variables); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func buildChart(lang string, netWorthHistoryData []domain.NetWorthHistory) Chart {
	if len(netWorthHistoryData) == 0 {
		return Chart{}
	}

	println(fmt.Sprintf("netWorthHistoryData: %+v", netWorthHistoryData))

	labelsDate := netWorthHistoryData[0].Dates
	labels := make([]string, len(labelsDate))
	for i, date := range labelsDate {
		labels[i] = monday.Format(date, "Jan", monday.Locale(lang))
	}

	series := make([]Series, 0, len(netWorthHistoryData))
	for _, netWorthHistory := range netWorthHistoryData {
		values := make([]float64, len(netWorthHistory.Amounts))
		for i, amount := range netWorthHistory.Amounts {
			f, err := strconv.ParseFloat(amount.Number(), 64)
			if err != nil {
				continue
			}
			values[i] = f
		}

		series = append(series, Series{
			Name: netWorthHistory.Name,
			Data: values,
		})
	}

	netWorthHistory := Chart{
		Options: &Options{
			Labels: labels,
		},
		Series: series,
	}

	println(fmt.Sprintf("netWorthHistory: %+v", netWorthHistory))

	return netWorthHistory
}
