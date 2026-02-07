package internal

import (
	"context"
	"embed"
	"log/slog"
	"net/http"

	"github.com/abiosoft/mold"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	slogchi "github.com/samber/slog-chi"
	"github.com/syrm/maille/internal/processor"
	"github.com/syrm/maille/internal/web"
)

//go:embed template/*.html
var dir embed.FS

func MaxBytesMiddleware(maxBytes int64) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

func Run(ctx context.Context, getenv func(string) string, pool *pgxpool.Pool, logger *slog.Logger) error {
	engine, errMold := mold.New(dir)

	if errMold != nil {
		return errMold
	}

	ofxParser := processor.OFXParser{}

	proc := processor.Processor{
		Pool: pool,
	}

	upload := web.Upload{
		Processor: proc,
		Parser:    ofxParser,
		Engine:    engine,
		Logger:    logger,
	}

	r := chi.NewRouter()
	r.Use(MaxBytesMiddleware(200 * 1024 * 1024))
	r.Use(slogchi.New(logger))
	r.Mount("/", upload.GetRouter())
	err := http.ListenAndServe(":13000", r)

	return err
}
