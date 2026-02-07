package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/syrm/maille/internal"

	"github.com/google/gops/agent"
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

	err := internal.Run(ctx, os.Getenv, pool, logger)
	if err != nil {
		logger.ErrorContext(ctx, "failed to start server", slog.Any("error", err))
	}
}
