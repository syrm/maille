package web

import (
	"context"
	"io"
	"log/slog"
	"net/http"

	"github.com/abiosoft/mold"
	"github.com/go-chi/chi/v5"
	"github.com/syrm/maille/internal"
)

type TransactionParser interface {
	Parse(ctx context.Context, reader io.Reader, batchSize int, fn func(context.Context, string, []internal.Transaction) error) error
}

type TransactionStore interface {
	Process(ctx context.Context, currency string, stmts []internal.Transaction) error
}

type Upload struct {
	Parser TransactionParser
	Store  TransactionStore
	Engine mold.Engine
	Logger *slog.Logger
}

func (u Upload) Router() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/upload", u.Get)
	r.Post("/upload", u.Post)

	return r
}

func (u Upload) Get(w http.ResponseWriter, r *http.Request) {
	errRender := u.Engine.Render(w, "template/upload.html", nil)

	if errRender != nil {
		u.Logger.ErrorContext(r.Context(), "failed to render", slog.Any("error", errRender))
	}
}

func (u Upload) Post(w http.ResponseWriter, r *http.Request) {
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
		w.Write([]byte("Good 3"))
		return
		// @TODO redirection
	}

	errParse := u.Parser.Parse(r.Context(), file, 200_000, u.Store.Process)

	if errParse != nil {
		u.Logger.ErrorContext(r.Context(), "failed to read all file", slog.Any("error", errParse))
		w.Write([]byte("Good 5"))
		return
		// @TODO redirection
	}

	w.Write([]byte("Good"))
}
