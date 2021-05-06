package keys

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type contextKey string

type JWTValidator func(req *http.Request, claims Claims) error
type JWTEnabler func(req *http.Request) bool

const RequestSubjectKey = contextKey("subject")
const RequestClaimsKey = contextKey("claims")

var AlwaysEnabled = func(req *http.Request) bool { return true }
var NoExtraValidation = func(req *http.Request, claims Claims) error { return nil }

func ValidateAppID(appID string) JWTValidator {
	return func(req *http.Request, claims Claims) error {
		if claims.AppID != appID {
			return fmt.Errorf("invalid appID: got %s, wanted %s", claims.AppID, appID)
		}
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
			if enabler(r) {
				token := r.Header["Authorization"]
				if len(token) == 0 {
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

				err = validator(r, t)
				if err != nil {
					http.Error(w, err.Error(), 403)
					return
				}

				ctx = context.WithValue(ctx, RequestSubjectKey, t.Subject)
				ctx = context.WithValue(ctx, RequestClaimsKey, t)
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
