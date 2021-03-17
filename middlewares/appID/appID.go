package appID

import (
	"context"
	"net/http"
	"strings"

	httpmdlwr "goa.design/goa/v3/http/middleware"
)

type contextKey string

const RequestAppIDKey = contextKey("appID")

// RequestAppID adds a appID struct to the context, to be shared by the chain of middlewares
func RequestAppID() func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			rw := httpmdlwr.CaptureResponse(w)
			if r.Method == "OPTIONS" {
				h.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			if strings.Contains(r.URL.Path, "/api/") {
				appID := "[missing]"
				if _, ok := r.Header["X-App-Id"]; ok && len(r.Header["X-App-Id"]) > 0 {
					appID = r.Header["X-App-Id"][0]
				}
				ctx = context.WithValue(ctx, RequestAppIDKey, appID)
			}

			h.ServeHTTP(rw, r.WithContext(ctx))
		})
	}
}

func ContextAppID(ctx context.Context) string {
	if appID, ok := ctx.Value(RequestAppIDKey).(string); ok {
		return appID
	}

	return ""
}
