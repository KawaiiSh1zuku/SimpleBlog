package repository

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"strings"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
)

type SlowQueryTracer struct {
	Threshold time.Duration
	Now       func() time.Time
}

type queryTrace struct {
	startedAt   time.Time
	fingerprint string
}

type queryTraceContextKey struct{}

func NewSlowQueryTracer(threshold time.Duration) *SlowQueryTracer {
	return &SlowQueryTracer{Threshold: threshold, Now: time.Now}
}

func (tracer *SlowQueryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	normalized := strings.Join(strings.Fields(data.SQL), " ")
	hash := sha256.Sum256([]byte(normalized))
	return context.WithValue(ctx, queryTraceContextKey{}, queryTrace{
		startedAt: tracer.Now(), fingerprint: fmt.Sprintf("%x", hash[:8]),
	})
}

func (tracer *SlowQueryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	trace, ok := ctx.Value(queryTraceContextKey{}).(queryTrace)
	if !ok {
		return
	}
	duration := tracer.Now().Sub(trace.startedAt)
	if duration < tracer.Threshold {
		return
	}
	slog.WarnContext(ctx, "slow PostgreSQL query",
		"request_id", chimiddleware.GetReqID(ctx),
		"query_fingerprint", trace.fingerprint,
		"duration_ms", duration.Milliseconds(),
		"command", data.CommandTag.String(),
		"sqlstate", SQLState(data.Err),
	)
}
