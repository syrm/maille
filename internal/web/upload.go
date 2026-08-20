package web

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"github.com/CloudyKit/jet/v6"
	"github.com/go-chi/chi/v5"

	"github.com/syrm/maille/internal"
	"github.com/syrm/maille/internal/middleware"
)

const maxUploadBytes = 100 << 20

type transactionImporter interface {
	Import(context.Context, io.Reader) (int, error)
}

type transactionClassifier interface {
	Classify(context.Context) error
}

type Upload struct {
	Renderer   Renderer
	Importer   transactionImporter
	Classifier transactionClassifier
	Logger     *slog.Logger
}

func (u Upload) Router() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", u.Get)
	r.Post("/", u.Post)

	return r
}

func (u Upload) Get(w http.ResponseWriter, r *http.Request) {
	variables := jet.VarMap{}
	variables.Set("lang", r.Context().Value(middleware.LangKey))
	variables.Set("pageTitle", "Importer un relevé")
	variables.Set("currentPage", "upload")
	variables.Set("errorMessage", uploadErrorMessage(r.URL.Query().Get("error")))

	if err := u.Renderer.Render(w, "upload", variables); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (u Upload) Post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)

	if err := r.ParseMultipartForm(8 << 20); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			redirectUploadError(w, r, "too-large")
			return
		}
		redirectUploadError(w, r, "invalid-upload")
		return
	}

	file, _, errFile := r.FormFile("file")
	if errFile != nil {
		redirectUploadError(w, r, "missing-file")
		return
	}
	defer file.Close()

	count, errImport := u.Importer.Import(ctx, file)
	if errImport != nil {
		u.Logger.ErrorContext(r.Context(), "failed to import file", slog.Any("error", errImport))
		if errors.Is(errImport, internal.ErrDuplicateImport) {
			redirectUploadError(w, r, "duplicate")
			return
		}
		redirectUploadError(w, r, "import-failed")
		return
	}

	errClass := u.Classifier.Classify(ctx)
	if errClass != nil {
		u.Logger.ErrorContext(r.Context(), "failed to classify transaction", slog.Any("error", errClass))
		redirectDashboard(w, r, count, true)
		return
	}

	redirectDashboard(w, r, count, false)
}

func redirectUploadError(w http.ResponseWriter, r *http.Request, code string) {
	http.Redirect(w, r, "/upload?error="+url.QueryEscape(code), http.StatusSeeOther)
}

func redirectDashboard(w http.ResponseWriter, r *http.Request, count int, classificationWarning bool) {
	query := url.Values{
		"status": {"imported"},
		"count":  {strconv.Itoa(count)},
	}
	if classificationWarning {
		query.Set("warning", "classification")
	}
	http.Redirect(w, r, "/?"+query.Encode(), http.StatusSeeOther)
}

func uploadErrorMessage(code string) string {
	switch code {
	case "too-large":
		return "Le fichier dépasse la taille maximale de 100 Mo."
	case "invalid-upload":
		return "Le formulaire d’import est invalide. Veuillez réessayer."
	case "missing-file":
		return "Sélectionnez un fichier OFX avant de continuer."
	case "duplicate":
		return "Ce relevé a déjà été importé. Aucune transaction n’a été ajoutée."
	case "import-failed":
		return "Le fichier OFX n’a pas pu être importé. Vérifiez son format et le compte bancaire associé."
	default:
		return ""
	}
}
