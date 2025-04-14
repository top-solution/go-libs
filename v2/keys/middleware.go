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

const Anonymous = "Anonymous"

// EnableWithClaims is a middleware that enables the JWT authentication for a specific path prefix
func EnableWithPrefix(prefix string) Option {
	return func(h http.Handler, w http.ResponseWriter, r *http.Request, _ Claims, beforeAuth bool) (bool, error) {
		if beforeAuth && !strings.HasPrefix(r.URL.Path, prefix) {
			h.ServeHTTP(w, r)
			return false, nil
		}
		return true, nil
	}
}

// DisableWithClaims is a middleware that disables the JWT authentication for a specific path prefix
func DisableWithPrefix(prefix string) Option {
	return func(h http.Handler, w http.ResponseWriter, r *http.Request, _ Claims, beforeAuth bool) (bool, error) {
		if beforeAuth && strings.HasPrefix(r.URL.Path, prefix) {
			return false, nil
		}
		return true, nil
	}
}

// Passthrough is a middleware that allows requests without authentication to pass through, setting the subject to Anonymous
// WARNING: using this means you need to handle the authorization yourself
func Passthrough() Option {
	return func(h http.Handler, w http.ResponseWriter, r *http.Request, _ Claims, _ bool) (bool, error) {
		_, err := getToken(r)
		// Allow requests without authentication to pass through
		if err != nil {
			ctx := context.WithValue(r.Context(), RequestSubjectKey, Anonymous)
			h.ServeHTTP(w, r.WithContext(ctx))
			return false, nil
		}
		return true, nil
	}
}

// Option is a function that can be used to configure the middleware
// It returns a boolean indicating if the request should actually be processed
type Option func(h http.Handler, w http.ResponseWriter, r *http.Request, claims Claims, beforeAuth bool) (cont bool, err error)

func RequestJWT(keys *JWT, opts ...Option) func(http.Handler) http.Handler {
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

			// Run the options before authentication, so they can decide to skip it
			for _, opt := range opts {
				cont, err := opt(h, w, r, Claims{}, true)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if !cont {
					return
				}
			}

			token, err := getToken(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}

			t, err := keys.ParseAndValidateToken(token)
			if err != nil {
				if errors.Is(err, ErrInvalidToken) {
					http.Error(w, err.Error(), http.StatusForbidden)
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			ctx = context.WithValue(ctx, RequestSubjectKey, t.Subject)
			ctx = context.WithValue(ctx, RequestClaimsKey, t)
			ctx = context.WithValue(ctx, RequestTokenKey, token)

			// Run the options after authentication
			for _, opt := range opts {
				cont, err := opt(h, w, r, t, false)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if !cont {
					return
				}
			}

			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func SubjectFromContext(ctx context.Context) string {
	if elem, ok := ctx.Value(RequestSubjectKey).(string); ok {
		return elem
	}

	return Anonymous
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

func getToken(r *http.Request) (string, error) {
	token := r.Header["Authorization"]
	if len(token) == 0 {
		return "", errors.New("missing authorization header")
	}
	tok := token[0]
	if !strings.HasPrefix(strings.ToLower(tok), "bearer ") {
		return "", errors.New("invalid authorization header")
	}

	return strings.Split(tok, " ")[1], nil
}
