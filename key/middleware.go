package key

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const RequestUsernameKey = contextKey("username")

// RequestKey adds a appID struct to the context, to be shared by the chain of middlewares
func RequestKey(keys *JWT, enabled func(req *http.Request) bool) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			if r.Method == "OPTIONS" {
				h.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			if enabled(r) {
				token := r.Header["X-Authorization"]
				if token == nil || len(token) == 0 {
					http.Error(w, "Missing jwt token", 401)
					return
				}
				tok := token[0]
				if !strings.HasPrefix(strings.ToLower(tok), "bearer ") {
					http.Error(w, "Token not valid", 401)
					return
				}

				tok = strings.Split(tok, " ")[1]

				t, err := keys.GetParsedToken(tok)
				if err != nil {
					http.Error(w, err.Error(), 500)
					return
				}
				ctx = context.WithValue(ctx, RequestUsernameKey, t[string(RequestUsernameKey)])
			}

			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
