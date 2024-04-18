package middlewares

import (
	"context"

	"github.com/top-solution/go-libs/ctxlog"

	"github.com/top-solution/go-libs/middlewares/meta"
	goa "goa.design/goa/v3/pkg"
)

func DefaultEndpointMiddlewares(endpoints Endpoints) Endpoints {
	endpoints.Use(LogError())
	endpoints.Use(LogStartEndpoint(NeverCondition))
	endpoints.Use(RequestMetaEndpoint())

	return endpoints
}

// RequestMetaEndpoint is a middleware endpoint that augments the request meta with info about
// the method, payload and service
func RequestMetaEndpoint() func(goa.Endpoint) goa.Endpoint {
	return func(e goa.Endpoint) goa.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			meta, ok := meta.ContextMeta(ctx)
			if !ok {
				panic("metadata not found in context. Have you setup the RequestMeta middleware?")
			}

			meta.Method = ctx.Value(goa.MethodKey).(string)
			meta.Service = ctx.Value(goa.ServiceKey).(string)
			meta.Payload = req

			res, err := e(ctx, req)

			return res, err
		}
	}
}

// LogStartEndpoint logs the request start, and enrichs the logger with info about the request.
// note that this middleware only runs if a matching method exists, and the request didn't fail its validation
func LogStartEndpoint(shouldLogFunc MetaCondition) func(goa.Endpoint) goa.Endpoint {
	return func(e goa.Endpoint) goa.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			meta, _ := meta.ContextMeta(ctx)

			if ShouldLogPayloads && meta.Service != "" {
				ctx = ctxlog.WithFields(ctx, map[string]interface{}{"method": meta.Method, "payload": meta.Payload, "service": meta.Service})
			}

			if !shouldLog(meta, shouldLogFunc) {
				ctxlog.Info(ctx, "Request start", "action", "start")
			}

			res, err := e(ctx, req)

			return res, err
		}
	}
}

// LogError is a middleware that logs errors which result in an untracked error
func LogError() func(goa.Endpoint) goa.Endpoint {
	return func(e goa.Endpoint) goa.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			res, err := e(ctx, req)
			if err != nil {
				e, ok := err.(*goa.ServiceError)
				if !ok || e.Fault {
					ctxlog.Error(ctx, "Request failed", "err", err)
				}
			}

			return res, err
		}
	}
}
