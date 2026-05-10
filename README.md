# T-SLOG

[![Go](https://github.com/tinh-tinh/tslog/actions/workflows/go.yml/badge.svg)](https://github.com/tinh-tinh/tslog/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/tinh-tinh/tslog/graph/badge.svg?token=YOUR_TOKEN)](https://codecov.io/gh/tinh-tinh/tslog)
[![Go Reference](https://pkg.go.dev/badge/github.com/tinh-tinh/tslog.svg)](https://pkg.go.dev/github.com/tinh-tinh/tslog)
[![Go Report Card](https://goreportcard.com/badge/github.com/tinh-tinh/tslog)](https://goreportcard.com/report/github.com/tinh-tinh/tslog)
![GitHub License](https://img.shields.io/github/license/tinh-tinh/tslog)

**tslog** is a structured logging module for the [Tinh Tinh](https://github.com/tinh-tinh/tinhtinh) framework, built on top of Go's standard [`log/slog`](https://pkg.go.dev/log/slog) package. It integrates `slog` as a first-class provider within the Tinh Tinh dependency injection system and exposes a flexible middleware for HTTP request logging.

---

## Features

- 🔌 **DI-friendly** — Register any `slog.Handler` as a module-level provider using `ForRoot`
- 💉 **Easy injection** — Retrieve the logger anywhere via `Inject(module)`
- 🛣️ **HTTP middleware** — Log every incoming request with a customisable function
- ⏭️ **Skip paths** — Exclude health-check or any other routes from logging
- 🔗 **Context-aware** — Use `slog.Logger.InfoContext` together with request context for trace/request-ID propagation

---

## Requirements

| Dependency | Version |
|---|---|
| Go | ≥ 1.24 |
| `github.com/tinh-tinh/tinhtinh` | v2 |

---

## Installation

```bash
go get github.com/tinh-tinh/tslog
```

---

## Quick Start

### 1. Register the module

Pass any standard `slog.Handler` to `ForRoot`. The logger is registered as a named provider (`TSLOG`) and made available across all imported modules.

```go
package main

import (
    "log/slog"
    "os"

    "github.com/tinh-tinh/tinhtinh/v2/core"
    "github.com/tinh-tinh/tslog"
)

func main() {
    appModule := func() core.Module {
        return core.NewModule(core.NewModuleOptions{
            Imports: []core.Modules{
                tslog.ForRoot(slog.NewJSONHandler(os.Stdout, nil)),
            },
            Controllers: []core.Controllers{userController},
        })
    }

    app := core.CreateFactory(appModule)
    app.Listen(3000)
}
```

### 2. Inject and use the logger

Inside any controller or service, call `tslog.Inject(module)` to obtain the `*slog.Logger`.

```go
func userController(module core.Module) core.Controller {
    ctrl := module.NewController("users")
    logger := tslog.Inject(module)

    ctrl.Get("", func(ctx core.Ctx) error {
        logger.Info("Request processed",
            "http", slog.Group("request",
                "method", "GET",
                "path", "/api/users",
                "status", 200,
            ),
            "duration_ms", 42,
        )
        return ctx.JSON(map[string]any{"ok": true})
    })

    return ctrl
}
```

---

## HTTP Middleware

### Basic logging middleware

`tslog.Middleware` wraps a user-supplied function (`Fnc`) that receives the request context. Use it to log method, path, latency, or any attribute you need.

```go
loggerMiddleware := tslog.Middleware(tslog.MiddlewareOptions{
    Fnc: func(ctx core.Ctx) {
        logger := ctx.Ref(tslog.TSLOG).(*slog.Logger)
        logger.Info("Incoming request",
            slog.Group("http",
                slog.Group("request",
                    "method", ctx.Req().Method,
                    "path",   ctx.Req().URL.Path,
                ),
            ),
        )
    },
})

appModule := func() core.Module {
    return core.NewModule(core.NewModuleOptions{
        Imports:     []core.Modules{tslog.ForRoot(slog.NewJSONHandler(os.Stdout, nil))},
        Controllers: []core.Controllers{userController},
        Middlewares: []core.Middleware{loggerMiddleware},
    })
}
```

### Skipping paths

Supply `SkipPaths` to prevent certain routes (e.g., health-check endpoints) from being logged.

```go
loggerMiddleware := tslog.Middleware(tslog.MiddlewareOptions{
    SkipPaths: []string{"/health", "/readyz"},
    Fnc: func(ctx core.Ctx) {
        logger := ctx.Ref(tslog.TSLOG).(*slog.Logger)
        logger.Info("Incoming request",
            "method", ctx.Req().Method,
            "path",   ctx.Req().URL.Path,
        )
    },
})
```

---

## Context-Aware Logging (Trace / Request ID)

Combine tslog with a custom `slog.Handler` that reads values from the request context to automatically attach trace or request IDs to every log line.

```go
const reqIDKey = "req_id"

// ContextHandler enriches each log record with the request ID stored in ctx.
type ContextHandler struct{ slog.Handler }

func (h ContextHandler) Handle(ctx context.Context, r slog.Record) error {
    if reqID, ok := ctx.Value(reqIDKey).(string); ok {
        r.AddAttrs(slog.String("req_id", reqID))
    }
    return h.Handler.Handle(ctx, r)
}

// traceMiddleware assigns a unique request ID and stores it in context.
traceMiddleware := func(ctx core.Ctx) error {
    ctx.Set(reqIDKey, rand.Text())
    return ctx.Next()
}

// loggerMiddleware logs using the enriched context.
loggerMiddleware := tslog.Middleware(tslog.MiddlewareOptions{
    Fnc: func(ctx core.Ctx) {
        logger := ctx.Ref(tslog.TSLOG).(*slog.Logger)
        logger.InfoContext(ctx.Req().Context(), "API Request")
    },
})

appModule := func() core.Module {
    base := slog.NewJSONHandler(os.Stdout, nil)
    return core.NewModule(core.NewModuleOptions{
        Imports:     []core.Modules{tslog.ForRoot(ContextHandler{Handler: base})},
        Controllers: []core.Controllers{userController},
        Middlewares: []core.Middleware{traceMiddleware, loggerMiddleware},
    })
}
```

Every log line produced inside a request will now contain a `req_id` field automatically.

---

## API Reference

### `ForRoot(h slog.Handler) core.Modules`

Registers a new `*slog.Logger` (backed by `h`) as the `TSLOG` provider in the module. Call this once in your root module's `Imports`.

### `Inject(module core.Module) *slog.Logger`

Retrieves the `*slog.Logger` registered by `ForRoot`. Returns `nil` if `ForRoot` was not imported.

### `Middleware(options MiddlewareOptions) core.Middleware`

Returns a Tinh Tinh middleware that:
1. Skips execution for any path listed in `SkipPaths`.
2. Calls `Fnc(ctx)` for all other requests.

#### `MiddlewareOptions`

| Field | Type | Description |
|---|---|---|
| `SkipPaths` | `[]string` | URL paths that bypass the middleware |
| `Fnc` | `func(ctx core.Ctx)` | Logging logic executed per request |

### `TSLOG`

A `core.Provide` constant (`"TS_LOG"`) used as the DI token. Use `ctx.Ref(tslog.TSLOG)` inside middleware to retrieve the logger.

---

## Testing

```bash
go test -cover ./...
```

The CI pipeline runs tests against Go **1.24**, **1.25**, and **1.26**, with coverage reports uploaded to [Codecov](https://codecov.io/gh/tinh-tinh/tslog).

---

## License

Distributed under the MIT License. See [LICENSE](LICENSE) for details.
