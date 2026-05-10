package tslog_test

import (
	"context"
	"crypto/rand"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-openapi/testify/require"
	"github.com/tinh-tinh/tinhtinh/v2/core"
	"github.com/tinh-tinh/tslog"
)

func TestMiddleware(t *testing.T) {
	loggerMiddleware := tslog.Middleware(tslog.MiddlewareOptions{
		Fnc: func(ctx core.Ctx) {
			logger := ctx.Ref(tslog.TSLOG).(*slog.Logger)
			logger.Info("Middleware executed",
				slog.Group("http",
					slog.Group("request",
						"method", ctx.Req().Method,
						"path", ctx.Req().URL.Path,
					),
				),
			)
		},
		SkipPaths: []string{
			"/test/health",
		},
	})

	ctrlFnc := func(module core.Module) core.Controller {
		ctrl := module.NewController("test")

		ctrl.Get("", func(ctx core.Ctx) error {

			return ctx.JSON(true)
		})

		ctrl.Get("/health", func(ctx core.Ctx) error {
			return ctx.JSON(true)
		})

		return ctrl
	}

	moduleFnc := func() core.Module {
		return core.NewModule(core.NewModuleOptions{
			Imports:     []core.Modules{tslog.ForRoot(slog.NewJSONHandler(os.Stdout, nil))},
			Controllers: []core.Controllers{ctrlFnc},
			Middlewares: []core.Middleware{loggerMiddleware},
		})
	}
	app := core.CreateFactory(moduleFnc)

	testServer := httptest.NewServer(app.PrepareBeforeListen())
	defer testServer.Close()

	testClient := testServer.Client()

	resp, err := testClient.Get(testServer.URL + "/test")
	require.Nil(t, err)
	defer resp.Body.Close()

	resp, err = testClient.Get(testServer.URL + "/test/health")
	require.Nil(t, err)
	defer resp.Body.Close()
}

const reqIDKey = "req_id"

type ContextHandler struct {
	slog.Handler
}

// 3. Implement the Handle method
func (h ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	// Extract the value from the context
	if reqID, ok := ctx.Value(reqIDKey).(string); ok {
		// Add the context value as an attribute to the log record
		r.AddAttrs(slog.String("req_id", reqID))
	}

	// Pass the modified record to the underlying handler
	return h.Handler.Handle(ctx, r)
}

func TestMiddlewareContext(t *testing.T) {

	loggerMiddleware := tslog.Middleware(tslog.MiddlewareOptions{
		Fnc: func(ctx core.Ctx) {
			logger := ctx.Ref(tslog.TSLOG).(*slog.Logger)

			context := ctx.Req().Context()
			logger.InfoContext(context, "API Request")
		},
	})

	traceMiddleware := func(ctx core.Ctx) error {
		ctx.Set(reqIDKey, rand.Text())
		return ctx.Next()
	}

	ctrlFnc := func(module core.Module) core.Controller {
		ctrl := module.NewController("test")

		ctrl.Get("", func(ctx core.Ctx) error {
			return ctx.JSON(true)
		})

		return ctrl
	}

	moduleFnc := func() core.Module {
		baseHandler := slog.NewJSONHandler(os.Stdout, nil)

		return core.NewModule(core.NewModuleOptions{
			Imports:     []core.Modules{tslog.ForRoot(ContextHandler{Handler: baseHandler})},
			Controllers: []core.Controllers{ctrlFnc},
			Middlewares: []core.Middleware{traceMiddleware, loggerMiddleware},
		})
	}
	app := core.CreateFactory(moduleFnc)

	testServer := httptest.NewServer(app.PrepareBeforeListen())
	defer testServer.Close()

	testClient := testServer.Client()

	resp, err := testClient.Get(testServer.URL + "/test")
	require.Nil(t, err)
	defer resp.Body.Close()
}
