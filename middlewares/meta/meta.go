package meta

import (
	"context"
	"net/http"
	"time"

	httpmdlwr "goa.design/goa/v3/http/middleware"
)

type contextKey string

const RequestMetaKey = contextKey("request-meta")

type Meta struct {
	Duration time.Duration
	Method   string
	Payload  interface{}
	Service  string
	URL      string
	Verb     string
}

// RequestMeta adds a meta struct to the context, to be shared by the chain of middlewares
func RequestMeta() func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			rw := httpmdlwr.CaptureResponse(w)

			meta := &Meta{
				Verb: r.Method,
				URL:  r.URL.String(),
			}
			ctx = context.WithValue(ctx, RequestMetaKey, meta)

			h.ServeHTTP(rw, r.WithContext(ctx))
		})
	}
}

func ContextMeta(ctx context.Context) (*Meta, bool) {
	meta, ok := ctx.Value(RequestMetaKey).(*Meta)
	if meta == nil {
		meta = &Meta{}
	}

	return meta, ok
}
