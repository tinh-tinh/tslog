package tslog

import (
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/tinh-tinh/tinhtinh/v2/core"
)

type wrappedResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *wrappedResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.statusCode = statusCode
}

type MiddlewareOptions struct {
	SkipPaths []string
	Attrs     []slog.Attr
}

func Middleware(options MiddlewareOptions) func(ctx core.Ctx) error {
	return func(ctx core.Ctx) error {
		if slices.Contains(options.SkipPaths, ctx.Req().URL.Path) {
			return ctx.Next()
		}

		logger, ok := ctx.Ref(TSLOG).(*slog.Logger)
		if !ok {
			panic("tslog not found")
		}

		start := time.Now()
		wrapped := &wrappedResponseWriter{
			ResponseWriter: ctx.Res(),
			statusCode:     http.StatusOK,
		}
		ctx.SetCtx(wrapped, ctx.Req())
		ctx.Next()

		var level slog.Level
		if wrapped.statusCode >= 400 && wrapped.statusCode < 500 {
			level = slog.LevelWarn
		} else if wrapped.statusCode >= 500 {
			level = slog.LevelError
		} else {
			level = slog.LevelInfo
		}

		logAttrs := append(options.Attrs,
			slog.Int("status", wrapped.statusCode),
			slog.String("method", ctx.Req().Method),
			slog.String("path", ctx.Req().URL.Path),
			slog.Duration("duration", time.Since(start)),
		)
		logger.LogAttrs(ctx.Req().Context(), level, "HTTP request", logAttrs...)

		return nil
	}
}
