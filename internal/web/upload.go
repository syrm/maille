package web

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/trace"

	"github.com/syrm/maille/internal"
)

type Upload struct {
	Renderer         Renderer
	AccountStore     internal.AccountStore
	TransactionStore internal.TransactionStore
	Importer         internal.Importer
	Classifier       internal.Classifier
	Tracer           trace.Tracer
	Logger           *slog.Logger
}

func (u Upload) Router() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", u.Get)
	r.Post("/", u.Post)

	return r
}

func (u Upload) Get(w http.ResponseWriter, r *http.Request) {
	if err := u.Renderer.Render(w, "upload", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (u Upload) Post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	err := r.ParseMultipartForm(100 << 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		w.Write([]byte("Good 1"))
		return
	}

	errForm := r.ParseForm()

	if errForm != nil {
		u.Logger.ErrorContext(r.Context(), "failed to parse form", slog.Any("error", errForm))
		w.Write([]byte("Good 2"))
		return
		// @TODO redirection
	}

	file, _, errFile := r.FormFile("file")
	_ = file
	_ = errFile

	if errFile != nil {
		u.Logger.ErrorContext(r.Context(), "failed to read file form", slog.Any("error", errForm))
		w.Write([]byte("pas good 2"))
		return
		// @TODO redirection
	}

	errImport := u.Importer.Import(ctx, file)
	if errImport != nil {
		u.Logger.ErrorContext(r.Context(), "failed to import file", slog.Any("error", errImport))
		w.Write([]byte("Good 5"))
		return
		// @TODO redirection
	}

	errClass := u.Classifier.Classify(ctx)
	if errClass != nil {
		u.Logger.ErrorContext(r.Context(), "failed to classify transaction", slog.Any("error", errClass))
		w.Write([]byte("pas good"))
		return
		// @TODO redirection
	}

	w.Write([]byte("Good"))
}
