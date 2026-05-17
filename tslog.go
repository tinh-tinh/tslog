package tslog

import (
	"log/slog"

	"github.com/tinh-tinh/tinhtinh/v2/core"
)

const TSLOG core.Provide = "TS_LOG"

func ForRoot(h slog.Handler) core.Modules {
	return func(module core.Module) core.Module {
		module.NewProvider(core.ProviderOptions{
			Name:   TSLOG,
			Value:  slog.New(h),
			Status: core.PUBLIC,
		})

		return module
	}
}

func Inject(module core.Module) *slog.Logger {
	tslog, ok := module.Ref(TSLOG).(*slog.Logger)
	if !ok {
		return nil
	}

	return tslog
}
