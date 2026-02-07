package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/abiosoft/mold"
	"github.com/go-chi/chi/v5"
	"github.com/google/gops/agent"
	"github.com/jackc/pgx/v5/pgxpool"
	slogchi "github.com/samber/slog-chi"

	"github.com/syrm/maille/internal/ofx"
	"github.com/syrm/maille/internal/postgres"
	"github.com/syrm/maille/internal/web"
)

func main() {
	ctx := context.Background()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	}))

	// @TODO quid opentelemetry/tracing ?

	if err := agent.Listen(agent.Options{}); err != nil {
		logger.ErrorContext(ctx, "error creating gops agent", slog.Any("error", err))
	}

	pool, errPool := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if errPool != nil {
		logger.ErrorContext(ctx, "failed to connect to database", slog.Any("error", errPool))
		os.Exit(1)
	}
	defer pool.Close()

	err := run(ctx, os.Getenv, pool, logger)
	if err != nil {
		logger.ErrorContext(ctx, "failed to start server", slog.Any("error", err))
	}
}

func run(
	ctx context.Context,
	getenv func(string) string,
	pool *pgxpool.Pool,
	logger *slog.Logger,
) error {
	engine, errMold := mold.New(web.TemplateFS)
	if errMold != nil {
		logger.ErrorContext(ctx, "failed to create template engine", slog.Any("error", errMold))
		os.Exit(1)
	}

	upload := web.Upload{
		Parser: &ofx.Parser{},
		Store:  &postgres.Store{Pool: pool},
		Engine: engine,
		Logger: logger,
	}

	r := chi.NewRouter()
	// r.Use(maxBytesMiddleware(200 * 1024 * 1024))
	r.Use(slogchi.New(logger))
	r.Mount("/", upload.Router())

	if err := http.ListenAndServe(":13000", r); err != nil {
		logger.ErrorContext(ctx, "failed to start server", slog.Any("error", err))
	}

	return nil
}

func maxBytesMiddleware(maxBytes int64) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
