package tslog_test

import (
	"context"
	"crypto/rand"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-openapi/testify/require"
	"github.com/tinh-tinh/tinhtinh/v2/core"
	"github.com/tinh-tinh/tslog"
)

func TestMiddleware(t *testing.T) {
	loggerMiddleware := tslog.Middleware(tslog.MiddlewareOptions{
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
	hostname, _ := os.Hostname()

	loggerMiddleware := tslog.Middleware(tslog.MiddlewareOptions{
		Attrs: []slog.Attr{
			slog.Int("pid", os.Getpid()),
			slog.String("hostname", hostname),
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

func TestMiddlewareError(t *testing.T) {
	loggerMiddleware := tslog.Middleware(tslog.MiddlewareOptions{})

	ctrlFnc := func(module core.Module) core.Controller {
		ctrl := module.NewController("test")

		ctrl.Get("error", func(ctx core.Ctx) error {
			return ctx.Status(http.StatusInternalServerError).JSON(true)
		})

		ctrl.Get("warning", func(ctx core.Ctx) error {
			return ctx.Status(http.StatusBadRequest).JSON(true)
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

	resp, err := testClient.Get(testServer.URL + "/test/error")
	require.Nil(t, err)
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	resp, err = testClient.Get(testServer.URL + "/test/warning")
	require.Nil(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestPanic(t *testing.T) {
	loggerMiddleware := tslog.Middleware(tslog.MiddlewareOptions{})

	ctrlFnc := func(module core.Module) core.Controller {
		ctrl := module.NewController("test")

		ctrl.Get("error", func(ctx core.Ctx) error {
			return ctx.Status(http.StatusInternalServerError).JSON(true)
		})

		ctrl.Get("warning", func(ctx core.Ctx) error {
			return ctx.Status(http.StatusBadRequest).JSON(true)
		})

		return ctrl
	}

	moduleFnc := func() core.Module {
		return core.NewModule(core.NewModuleOptions{
			Imports:     []core.Modules{},
			Controllers: []core.Controllers{ctrlFnc},
			Middlewares: []core.Middleware{loggerMiddleware},
		})
	}
	app := core.CreateFactory(moduleFnc)

	testServer := httptest.NewServer(app.PrepareBeforeListen())
	defer testServer.Close()

	testClient := testServer.Client()

	resp, err := testClient.Get(testServer.URL + "/test/error")
	require.Nil(t, err)
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}
