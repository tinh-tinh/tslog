package tslog

import (
	"github.com/tinh-tinh/tinhtinh/v2/core"
)

type MiddlewareOptions struct {
	SkipPaths []string
	Fnc       func(ctx core.Ctx)
}

func Middleware(options MiddlewareOptions) func(ctx core.Ctx) error {
	return func(ctx core.Ctx) error {
		for _, path := range options.SkipPaths {
			if ctx.Req().URL.Path == path {
				return nil
			}
		}
		options.Fnc(ctx)
		return nil
	}
}
