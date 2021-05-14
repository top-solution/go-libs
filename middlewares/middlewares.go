package middlewares

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"time"

	log "github.com/inconshreveable/log15"
	"gitlab.com/top-solution/go-libs/logging/ctxlog"
	"gitlab.com/top-solution/go-libs/middlewares/appID"
	"gitlab.com/top-solution/go-libs/middlewares/meta"
	goahttp "goa.design/goa/v3/http"
	httpmdlwr "goa.design/goa/v3/http/middleware"
	goamdlwr "goa.design/goa/v3/middleware"
	"goa.design/plugins/cors"
)

func DefaultMiddlewares(handler http.Handler, server Server) http.Handler {
	// Remember that they have to be in reverse order
	// Recover ensure that requests cannot panic
	handler = Recover(server.Enc)(handler)

	// Request duration middleware
	handler = Duration()(handler)

	// LogEnd logs the end of every request (except for the alive service)
	handler = LogEnd()(handler)

	// RequestID reads the request id from headers or generate a new one
	handler = RequestID()(handler)

	// CtxLogger adds the logger to the context in every request
	handler = CtxLogger(server.Log)(handler)

	// Vary adds the Vary:Origin header
	handler = Vary()(handler)

	// NoCache adds the Cache-Control headers
	handler = NoCache()(handler)

	// RequestMeta adds a shared meta struct in the chain of middlewares
	handler = meta.RequestMeta()(handler)

	// RequestMeta adds a shared meta struct in the chain of middlewares
	handler = appID.RequestAppID()(handler)

	return handler
}

// // Recover converts panics into logger errors
func Recover(enc func(context.Context, http.ResponseWriter) goahttp.Encoder) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if e := recover(); e != nil {
					var err error
					switch x := e.(type) {
					case string:
						err = errors.New(x)
					case error:
						err = errors.New(x.Error())
					default:
						err = errors.New("unknown panic")
					}

					stack := make([]byte, 1024*10)
					runtime.Stack(stack, false)

					ctx := ctxlog.WithField(r.Context(), "stack", strings.Trim(string(stack), "\x00"))
					ctxlog.Error(ctx, "err", e)

					encodeError := goahttp.ErrorEncoder(enc, nil)
					err = encodeError(ctx, w, err)
					if err != nil {
						ctxlog.Error(ctx, "err", err)
					}
				}
			}()
			h.ServeHTTP(w, r)
		})
	}
}

// Duration stores the request duration in the request meta
func Duration() func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			meta, ok := meta.ContextMeta(ctx)
			if !ok {
				panic("metadata not found in context. Have you setup the meta.RequestMeta middleware?")
			}
			started := time.Now()
			rw := httpmdlwr.CaptureResponse(w)
			h.ServeHTTP(rw, r.WithContext(ctx))
			meta.Duration = time.Since(started)
		})
	}
}

// LogEnd uses the ctxlogger to output info about the request
// It requires the meta middleware to work properly and retrieve the method and service fields
func LogEnd() func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			meta, _ := meta.ContextMeta(ctx)
			rw := httpmdlwr.CaptureResponse(w)
			h.ServeHTTP(rw, r)

			if meta.Service == "alive" && meta.Method == "alive" || meta.Service == "" {
				return
			}

			ctx = ctxlog.WithFields(ctx,
				"bytes", rw.ContentLength,
				"duration", meta.Duration.String(),
				"method", meta.Method,
				"service", meta.Service,
				"status", rw.StatusCode,
				"url", meta.URL,
				"verb", meta.Verb)
			ctxlog.Info(ctx, "action", "end")
		})
	}
}

// RequestID is a wrapper around the goa middleware with the same name,
// except it also augments the ctxlog with the request id
func RequestID() func(http.Handler) http.Handler {
	goaReqID := httpmdlwr.RequestID(
		httpmdlwr.UseXRequestIDHeaderOption(true),
		httpmdlwr.XRequestHeaderLimitOption(128),
	)

	return func(h http.Handler) http.Handler {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			reqID := ctx.Value(goamdlwr.RequestIDKey)

			ctx = ctxlog.WithField(ctx, "reqID", reqID)

			h.ServeHTTP(w, r.WithContext(ctx))
		})

		return goaReqID(handler)
	}
}

// CtxLogger augments the context object with a ctxlog entry
func CtxLogger(entry log.Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			meta, ok := meta.ContextMeta(ctx)
			if !ok {
				panic("metadata not found in context. Have you setup the meta.RequestMeta middleware?")
			}
			entry = entry.New("url", meta.URL, "verb", meta.Verb)

			ctx = context.WithValue(ctx, ctxlog.LogKey, entry)
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
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
func NoCache() func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")

			h.ServeHTTP(w, r)
		})
	}
}

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

			if cors.MatchOriginRegexp(origin, regex) {
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
