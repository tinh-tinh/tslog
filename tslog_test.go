package tslog_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-openapi/testify/require"
	"github.com/tinh-tinh/tinhtinh/v2/core"
	"github.com/tinh-tinh/tslog"
)

func createApp() *core.App {
	ctrlFnc := func(module core.Module) core.Controller {
		ctrl := module.NewController("test")
		logger := tslog.Inject(module)

		ctrl.Get("", func(ctx core.Ctx) error {
			logger.Info("Request processed",
				"http", slog.Group("request",
					"method", "GET",
					"path", "/api/users",
					"status", 200,
				),
				"duration_ms", 150,
			)
			return ctx.JSON(true)
		})
		return ctrl
	}

	moduleFnc := func() core.Module {
		return core.NewModule(core.NewModuleOptions{
			Imports:     []core.Modules{tslog.ForRoot(slog.NewJSONHandler(os.Stdout, nil))},
			Controllers: []core.Controllers{ctrlFnc},
		})
	}

	return core.CreateFactory(moduleFnc)
}

func TestForRoot(t *testing.T) {
	app := createApp()

	testServer := httptest.NewServer(app.PrepareBeforeListen())
	defer testServer.Close()

	testClient := testServer.Client()

	resp, err := testClient.Get(testServer.URL + "/test")
	require.Nil(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
