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
	r.Post("/", c.Post)

	return r
}

func (c Classifier) Get(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/rules", http.StatusSeeOther)
}

func (c Classifier) Post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	errClass := c.Classifier.Classify(ctx)
	if errClass != nil {
		c.Logger.ErrorContext(r.Context(), "failed to classify transaction", slog.Any("error", errClass))
		http.Redirect(w, r, "/rules?error=run-failed", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/rules?status=ran", http.StatusSeeOther)
}
