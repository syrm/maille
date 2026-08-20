package web

import (
	"context"
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
	BreakdownCategory(ctx context.Context) []domain.BreakdownCategory
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

type LineSeries struct {
	Name   string    `json:"name"`
	Data   []float64 `json:"data"`
	Color  *string   `json:"color,omitempty"`
	Area   bool      `json:"area"`
	Dashed bool      `json:"dashed"`
}

type LineOptions struct {
	Width  *float64 `json:"width,omitempty"`
	Height *float64 `json:"height,omitempty"`
	Labels []string `json:"labels,omitempty"`
	YTicks *int     `json:"yTicks,omitempty"`
	Smooth bool     `json:"smooth"`
	Points bool     `json:"points"`
	Colors []string `json:"colors,omitempty"`
}

type LineChart struct {
	Series  []LineSeries `json:"series"`
	Options *LineOptions `json:"options,omitempty"`
}

type PieSeries struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
	Color *string `json:"color,omitempty"`
}

type PieOptions struct {
	Width      *float64 `json:"width,omitempty"`
	Height     *float64 `json:"height,omitempty"`
	Donut      bool     `json:"donut"`
	DonutWidth *float64 `json:"donutWidth,omitempty"`
	PadAngle   *float64 `json:"padAngle,omitempty"`
	ShowLabel  bool     `json:"showLabel"`
	Colors     []string `json:"colors,omitempty"`
}

type PieChart struct {
	Slices  []PieSeries `json:"slices"`
	Options *PieOptions `json:"options,omitempty"`
}

func (d Dashboard) Get(w http.ResponseWriter, r *http.Request) {
	balanceSummary := d.Reporter.BalanceSummary(r.Context())
	recentTransactions := d.Reporter.RecentTransactions(r.Context())
	netWorthHistoryData := d.Reporter.NetWorthHistory(r.Context())
	breakdownCategoryData := d.Reporter.BreakdownCategory(r.Context())
	netWorthHistory := buildChartLine(r.Context().Value(middleware.LangKey).(string), netWorthHistoryData)
	breakdownCategory := buildChartPie(breakdownCategoryData)

	variables := jet.VarMap{}
	variables.Set("lang", r.Context().Value(middleware.LangKey))
	variables.Set("pageTitle", "Tableau de bord")
	variables.Set("currentPage", "dashboard")
	variables.Set("notice", importNotice(r))
	variables.Set("warning", importWarning(r))
	variables.Set("balanceSummary", balanceSummary)
	variables.Set("recentTransactions", recentTransactions)
	variables.Set("hasRecentTransactions", len(recentTransactions) > 0)
	variables.Set("netWorthHistory", netWorthHistory)
	variables.Set("hasNetWorthHistory", len(netWorthHistory.Series) > 0)
	variables.Set("breakdownCategory", breakdownCategory)
	variables.Set("breakdownCategories", breakdownCategoryData)
	variables.Set("hasBreakdownCategory", len(breakdownCategory.Slices) > 0)

	if err := d.Renderer.Render(w, "home", variables); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func importNotice(r *http.Request) string {
	if r.URL.Query().Get("status") != "imported" {
		return ""
	}
	count, err := strconv.Atoi(r.URL.Query().Get("count"))
	if err != nil || count < 1 {
		return "Import terminé."
	}
	if count == 1 {
		return "1 transaction importée avec succès."
	}
	return strconv.Itoa(count) + " transactions importées avec succès."
}

func importWarning(r *http.Request) string {
	if r.URL.Query().Get("warning") == "classification" {
		return "Les transactions ont été importées, mais leur catégorisation automatique doit être relancée."
	}
	return ""
}

func buildChartLine(lang string, netWorthHistoryData []domain.NetWorthHistory) LineChart {
	if len(netWorthHistoryData) == 0 {
		return LineChart{}
	}

	labelsDate := netWorthHistoryData[0].Dates
	labels := make([]string, len(labelsDate))
	for i, date := range labelsDate {
		labels[i] = monday.Format(date, "Jan", monday.Locale(lang))
	}

	series := make([]LineSeries, 0, len(netWorthHistoryData))
	hasData := false
	for _, netWorthHistory := range netWorthHistoryData {
		values := make([]float64, len(netWorthHistory.Amounts))
		for i, amount := range netWorthHistory.Amounts {
			f, err := strconv.ParseFloat(amount.Number(), 64)
			if err != nil {
				continue
			}
			values[i] = f
			hasData = hasData || f != 0
		}

		series = append(series, LineSeries{
			Name: netWorthHistory.Name,
			Data: values,
		})
	}
	if !hasData {
		return LineChart{}
	}

	netWorthHistory := LineChart{
		Options: &LineOptions{
			Labels: labels,
			Width:  new(712.),
			Height: new(240.),
		},
		Series: series,
	}

	// heu et le marshall ?

	return netWorthHistory
}

func buildChartPie(breakdownCategoryData []domain.BreakdownCategory) PieChart {
	if len(breakdownCategoryData) == 0 {
		return PieChart{}
	}

	slices := make([]PieSeries, 0, len(breakdownCategoryData))
	for _, breakdownCategory := range breakdownCategoryData {
		f, err := strconv.ParseFloat(breakdownCategory.Amount.Number(), 64)
		if err != nil {
			continue
		}

		slices = append(slices, PieSeries{
			Label: breakdownCategory.Name,
			Value: f,
		})
	}

	return PieChart{
		Slices: slices,
		Options: &PieOptions{
			Donut:  true,
			Width:  new(160.),
			Height: new(160.),
		},
	}
}
