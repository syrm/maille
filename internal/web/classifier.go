package web

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/trace"

	"github.com/syrm/maille/internal"
)

type Classifier struct {
	Classifier internal.Classifier
	Tracer     trace.Tracer
	Logger     *slog.Logger
}

func (c Classifier) Router() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", c.Get)

	return r
}

func (c Classifier) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	errClass := c.Classifier.Classify(ctx)
	if errClass != nil {
		c.Logger.ErrorContext(r.Context(), "failed to classify transaction", slog.Any("error", errClass))
		w.Write([]byte("pas good"))
		return
		// @TODO redirection
	}
}
