package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/abiosoft/mold"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/gops/agent"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riandyrn/otelchi"
	"github.com/samber/oops"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/syrm/maille/internal"
	"github.com/syrm/maille/internal/ofx"
	"github.com/syrm/maille/internal/postgres"
	"github.com/syrm/maille/internal/web"
)

func main() {
	ctx := context.Background()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	}))

	if err := agent.Listen(agent.Options{}); err != nil {
		logger.ErrorContext(ctx, "error creating gops agent", slog.Any("error", err))
	}

	err := run(ctx, os.Getenv, logger)
	if err != nil {
		logger.ErrorContext(ctx, "failed to start server", slog.Any("error", err))
	}
}

func run(
	ctx context.Context,
	getenv func(string) string,
	logger *slog.Logger,
) error {
	engine, errMold := mold.New(web.TemplateFS)
	if errMold != nil {
		logger.ErrorContext(ctx, "failed to create template engine", slog.Any("error", errMold))
		os.Exit(1)
	}

	tracerProvider, res, err := BuildTracerProvider(getenv("TRACING_ENDPOINT"))
	_ = res
	if err != nil {
		return oops.Wrapf(err, "error during initialization of tracerProvider")
	}

	defer func() {
		if errShutdown := tracerProvider.Shutdown(context.Background()); errShutdown != nil {
			slog.Error("Error shutting down tracer provider", slog.Any("error", errShutdown))
		}
	}()

	cfg, err := pgxpool.ParseConfig(getenv("DATABASE_URL"))
	if err != nil {
		return oops.Wrapf(err, "failed to create connection pool")
	}

	cfg.ConnConfig.Tracer = postgres.SQLTracer{Tracer: tracerProvider.GetTracer("postgres")}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return err
	}

	transactionStore, err := postgres.BuildTransaction(ctx, pool, tracerProvider.GetTracer("postgres.transaction"))
	if err != nil {
		return err
	}

	accountStore := postgres.BuildAccount(ctx, pool, tracerProvider.GetTracer("postgres.account"))
	bankAccountStore := postgres.BuildBankAccount(ctx, pool, tracerProvider.GetTracer("postgres.bankaccount"))
	currencyStore := postgres.BuildCurrency(ctx, pool, tracerProvider.GetTracer("postgres.currency"))

	parser := ofx.Parser{
		Tracer: tracerProvider.GetTracer("ofx.Parser"),
	}

	importer := internal.Importer{
		Parser:           parser,
		TransactionStore: transactionStore,
		AccountStore:     accountStore,
		BankAccountStore: bankAccountStore,
		CurrencyStore:    currencyStore,
		Tracer:           tracerProvider.GetTracer("importer"),
		Logger:           logger,
	}

	upload := web.Upload{
		Importer: importer,
		Engine:   engine,
		Tracer:   tracerProvider.GetTracer("maille-web"),
		Logger:   logger,
	}

	r := chi.NewRouter()
	r.Use(
		otelchi.Middleware("maille", otelchi.WithChiRoutes(r)),
	)
	r.Use(middleware.Recoverer)
	r.Mount("/", upload.Router())

	logger.InfoContext(ctx, "launch web server", slog.Any("port", 13000))

	if err := http.ListenAndServe(":13000", r); err != nil {
		logger.ErrorContext(ctx, "failed to start server", slog.Any("error", err))
	}

	return nil
}

type Provider struct {
	otelResource *sdkresource.Resource
	// metricExporter  *prometheus.Exporter
	tracingEndpoint string
	tracerProvider  *sdktrace.TracerProvider
}

func BuildTracerProvider(tracingEndpoint string) (Provider, *sdkresource.Resource, error) {
	extraResources, err := sdkresource.New(
		context.Background(),
		sdkresource.WithContainer(),
		sdkresource.WithHost(),
		sdkresource.WithAttributes(
			semconv.ServiceNameKey.String("web"),
		),
	)
	if err != nil {
		return Provider{}, extraResources, oops.Wrapf(err, "unable to create an otel resource")
	}

	resource, err := sdkresource.Merge(
		sdkresource.Default(),
		extraResources,
	)
	if err != nil {
		return Provider{}, extraResources, oops.Wrapf(err, "unable to merge otel resource")
	}

	provider := Provider{
		otelResource: resource,
		// metricExporter:  metricExporter,
		tracingEndpoint: tracingEndpoint,
	}

	tracerProvider, err := provider.getTracerProvider()
	if err != nil {
		return Provider{}, resource, oops.Wrapf(err, "unable to create tracer provider")
	}

	provider.tracerProvider = tracerProvider

	return provider, resource, nil
}

func (p *Provider) getTracerProvider() (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	traceExporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint(p.tracingEndpoint), // @TODO env variable
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, oops.Wrapf(err, "error during creation of traceExporter")
	}

	tracer := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(p.otelResource),
	)

	// meter := sdkmetric.NewMeterProvider(
	// 	sdkmetric.WithReader(p.metricExporter),
	// )

	otel.SetTracerProvider(tracer)
	// otel.SetMeterProvider(meter)

	// otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tracer, nil
}

func (p *Provider) Shutdown(ctx context.Context) error {
	return p.tracerProvider.Shutdown(ctx)
}

func (p *Provider) GetTracer(name string) trace.Tracer {
	return p.tracerProvider.Tracer(name)
}
