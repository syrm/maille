package web

import (
	"log/slog"
	"net/http"

	"github.com/abiosoft/mold"
	"github.com/go-chi/chi/v5"
	"github.com/syrm/maille/internal/processor"
)

type Upload struct {
	Processor processor.Processor
	Parser    processor.OFXParser
	Engine    mold.Engine
	Logger    *slog.Logger
}

func (u Upload) GetRouter() *chi.Mux {
	apiRouter := chi.NewRouter()
	apiRouter.Get("/upload", u.Get)
	apiRouter.Post("/upload", u.Post)

	return apiRouter
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
	// defer file.Close()

	// out, errCreate := os.CreateTemp(os.TempDir(), "upload-ofx-")
	// defer os.Remove(out.Name())

	// if errCreate != nil {
	// 	u.Logger.ErrorContext(r.Context(), "failed to read file form", slog.Any("error", errForm))
	// 	// @TODO redirection
	// }

	// io.Copy(out, file)

	// mf, _ := os.Open(out.Name())

	// reader := bufio.NewReaderSize(file, 64*1024)
	errParse := u.Parser.Parse(r.Context(), file, 100_000, u.Processor.Process)

	if errParse != nil {
		u.Logger.ErrorContext(r.Context(), "failed to read all file", slog.Any("error", errParse))
		w.Write([]byte("Good 5"))
		return
		// @TODO redirection
	}

	w.Write([]byte("Good"))
}
