package middlewares

import (
	"net/http"
	"regexp"
	"strings"
)

type RequestCondition func(r *http.Request) bool

var NeverCondition = func(r *http.Request) bool { return false }
var AlwaysCondition = func(r *http.Request) bool { return true }
var PrefixCondition = func(prefic string) func(r *http.Request) bool {
	return func(r *http.Request) bool { return strings.HasPrefix(r.URL.Path, prefic) }
}

// Vary is a middleware that adds a Vary header to every request
func Vary() func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Vary", w.Header().Get("Vary")+" Origin")

			h.ServeHTTP(w, r)
		})
	}
}

// NoCache is a middleware that adds NoCache headers to every request
func NoCache(shouldSkipCacheFunc RequestCondition) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if shouldSkipCacheFunc(r) {
				w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
				w.Header().Set("Pragma", "no-cache")
				w.Header().Set("Expires", "0")
			}

			h.ServeHTTP(w, r)
		})
	}
}

// Cors is a middleware that adds CORS headers to every request matching the given regex
func Cors(allowedOrigins string) func(http.Handler) http.Handler {
	regex := regexp.MustCompile(allowedOrigins)

	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				// Not a CORS request
				h.ServeHTTP(w, r)
				return
			}

			if regex.Match([]byte(origin)) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				if acrm := r.Header.Get("Access-Control-Request-Method"); acrm != "" {
					// We are handling a preflight request
					w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, POST, DELETE, PATCH, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Authorization, Origin, X-Requested-With, Content-Type, Accept, X-App-Id")
				}
			}

			h.ServeHTTP(w, r)
		})
	}
}
