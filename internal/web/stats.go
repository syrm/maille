package web

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/syrm/maille/internal/domain"
)

type Reporter interface {
	Report(context.Context) domain.Stats
}

type Stats struct {
	Reporter Reporter
}

func (s Stats) Router() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", s.Get)

	return r
}

func (s Stats) Get(w http.ResponseWriter, r *http.Request) {
	stats := s.Reporter.Report(r.Context())

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		slog.Error("failed to encode response", "err", err)
	}
}
