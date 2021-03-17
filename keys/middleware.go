package keys

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

type contextKey string

const RequestSubjectKey = contextKey("subject")

func RequestKey(keys *JWT) func(http.Handler) http.Handler {
	return RequestKeyConditionally(keys, func(req *http.Request) bool { return true })
}

func RequestKeyConditionally(keys *JWT, enabled func(req *http.Request) bool) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			if r.Method == "OPTIONS" {
				h.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			if enabled(r) {
				token := r.Header["Authorization"]
				if token == nil || len(token) == 0 {
					http.Error(w, "missing authorization header", 401)
					return
				}
				tok := token[0]
				if !strings.HasPrefix(strings.ToLower(tok), "bearer ") {
					http.Error(w, "invalid authorization header", 401)
					return
				}

				tok = strings.Split(tok, " ")[1]

				t, err := keys.ParseAndValidateToken(tok)
				if err != nil {
					if errors.Is(err, ErrInvalidToken) {
						http.Error(w, err.Error(), 403)
					}
					http.Error(w, err.Error(), 500)
					return
				}
				ctx = context.WithValue(ctx, RequestSubjectKey, t.Subject)
			}

			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func Subject(ctx context.Context) string {
	if elem, ok := ctx.Value(RequestSubjectKey).(string); ok {
		return elem
	}

	return "Anonymous"
}
