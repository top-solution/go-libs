package keys

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

type contextKey string

type JWTValidator func(req *http.Request, claims Claims) error
type JWTEnabler func(req *http.Request) bool

const RequestSubjectKey = contextKey("subject")
const RequestClaimsKey = contextKey("claims")
const RequestTokenKey = contextKey("token")

var AlwaysEnabled = func(req *http.Request) bool { return true }
var NoExtraValidation = func(req *http.Request, claims Claims) error { return nil }

func ValidateAppID(appID string) JWTValidator {
	return func(req *http.Request, claims Claims) error {
		return nil
	}
}

func EnableWithPrefix(prefix string) JWTEnabler {
	return func(req *http.Request) bool { return strings.HasPrefix(req.URL.Path, prefix) }
}

func RequestJWT(keys *JWT, validator JWTValidator, enabler JWTEnabler) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			if r.Method == "OPTIONS" {
				h.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			if r.RequestURI == "/version" {
				h.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			if enabler(r) {
				token := r.Header["Authorization"]
				if len(token) == 0 {
					http.Error(w, "missing authorization header", http.StatusUnauthorized)
					return
				}
				tok := token[0]
				if !strings.HasPrefix(strings.ToLower(tok), "bearer ") {
					http.Error(w, "invalid authorization header", http.StatusUnauthorized)
					return
				}

				tok = strings.Split(tok, " ")[1]

				t, err := keys.ParseAndValidateToken(tok)
				if err != nil {
					if errors.Is(err, ErrInvalidToken) {
						http.Error(w, err.Error(), http.StatusForbidden)
					}
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				err = validator(r, t)
				if err != nil {
					http.Error(w, err.Error(), http.StatusForbidden)
					return
				}

				ctx = context.WithValue(ctx, RequestSubjectKey, t.Subject)
				ctx = context.WithValue(ctx, RequestClaimsKey, t)
				ctx = context.WithValue(ctx, RequestTokenKey, token[0])
			}

			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func SubjectFromContext(ctx context.Context) string {
	if elem, ok := ctx.Value(RequestSubjectKey).(string); ok {
		return elem
	}

	return "Anonymous"
}

func ClaimsFromContext(ctx context.Context) Claims {
	if elem, ok := ctx.Value(RequestClaimsKey).(Claims); ok {
		return elem
	}

	return Claims{}
}

func TokenFromContext(ctx context.Context) string {
	if elem, ok := ctx.Value(RequestTokenKey).(string); ok {
		return elem
	}
	return ""
}
