package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type SQLNameType struct{}

var SQLName SQLNameType

type SQLTracer struct {
	Tracer trace.Tracer
}

func (s SQLTracer) TraceCopyFromStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceCopyFromStartData) context.Context {
	fmt.Printf("SQLTracer %+v\n", data)
	name, _ := ctx.Value(SQLName).(string)

	spanCtx, span := s.Tracer.Start(ctx, "SQL "+name)

	span.SetAttributes(attribute.String("sql", "COPY "+data.TableName.Sanitize()+" ("+strings.Join(data.ColumnNames, ", ")+") FROM STDIN BINARY"))

	return spanCtx
}

func (s SQLTracer) TraceCopyFromEnd(ctx context.Context, _ *pgx.Conn, _ pgx.TraceCopyFromEndData) {
	span := trace.SpanFromContext(ctx)
	span.End()
}

func (s SQLTracer) TraceQueryStart(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	fmt.Printf("SQLTracer %+v\n", data)
	name, _ := ctx.Value(SQLName).(string)
	spanCtx, span := s.Tracer.Start(ctx, "SQL "+name)

	span.SetAttributes(attribute.String("sql", data.SQL))

	return spanCtx
}

func (s SQLTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	span := trace.SpanFromContext(ctx)
	span.End()
}
