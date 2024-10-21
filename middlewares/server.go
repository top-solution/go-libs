package middlewares

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	fslib "github.com/top-solution/go-libs/fs"
	"github.com/top-solution/go-libs/middlewares/codec"
	"github.com/top-solution/go-libs/middlewares/meta"
	goahttp "goa.design/goa/v3/http"
	goa "goa.design/goa/v3/pkg"
)

type Server struct {
	Log *slog.Logger
	Dec func(r *http.Request) goahttp.Decoder
	Enc func(context.Context, http.ResponseWriter) goahttp.Encoder
	Mux goahttp.Muxer
	Eh  func(context.Context, http.ResponseWriter, error)
}

func New(entry *slog.Logger) Server {
	dec := goahttp.RequestDecoder
	enc := codec.ResponseEncoder // use custom encoder handling CSV

	mux := NewMuxer()
	mux.Handle("GET", "/alive", Alive())
	mux.Handle("OPTIONS", "/*", Alive())

	return Server{
		Log: entry,
		Mux: mux,
		Dec: dec,
		Enc: enc,
		Eh: func(ctx context.Context, w http.ResponseWriter, err error) {
			entry.Error("%w", err)
		},
	}
}

func (s Server) ListenGracefully(handler http.Handler, host string, port int) {
	srv := &http.Server{Addr: host + ":" + strconv.Itoa(port), Handler: handler}

	idleConnsClosed := make(chan struct{})

	s.Log.Info("Listen and gracefully shutdown on " + host + ":" + strconv.Itoa(port))

	go func() {
		sigint := make(chan os.Signal, 1)

		// interrupt signal sent from terminal
		signal.Notify(sigint, os.Interrupt)
		// sigterm signal sent from kubernetes
		signal.Notify(sigint, syscall.SIGTERM)

		<-sigint

		s.Log.Info("Start graceful shutdown. Waiting for current request to finish")
		// We received an interrupt signal, shut down.
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			s.Log.Error("HTTP server Shutdown", "err", err)
		}

		close(idleConnsClosed)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener:
		s.Log.Error("HTTP server ListenAndServe", "err", err)
	}

	<-idleConnsClosed

	s.Log.Info("Shutdown")
}

// AddWebappHandler is an helper that adds a filehandler to the GET / and GET /* routes
// The idea is to serve a web applicatiom through it, except for the path starting with prefixesToExclude
// It will use a FallbackFs to serve the specificied root file as a deafault asset
func (s Server) AddWebappHandler(fs fs.FS, rootfile string, prefixesToExclude ...string) {
	fileHandler := func(rw http.ResponseWriter, r *http.Request) {
		http.FileServer(http.FS(&fslib.FallbackFs{FS: fs, Fallback: rootfile})).ServeHTTP(rw, r)
	}

	webappHandler := func(rw http.ResponseWriter, r *http.Request) {
		exclude := false
		for _, p := range prefixesToExclude {
			if strings.HasPrefix(r.URL.Path, p) {
				exclude = true
				break
			}
		}
		if exclude {
			ctx := context.WithValue(r.Context(), goahttp.AcceptTypeKey, r.Header.Get("Accept"))
			enc := goahttp.ResponseEncoder(ctx, rw)
			rw.WriteHeader(http.StatusNotFound)
			err := enc.Encode(goahttp.NewErrorResponse(ctx, fmt.Errorf("404 page not found")))
			if err != nil {
				return
			}
		} else {
			fileHandler(rw, r)
		}
	}

	s.Mux.Handle("GET", "/*", webappHandler)
	s.Mux.Handle("GET", "/", webappHandler)
	s.Log.Info("Webapp is served served @ /", "verb", "GET")
}

type Endpoints interface {
	Use(m func(goa.Endpoint) goa.Endpoint)
}

func Alive() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		ctx := r.Context()

		meta, ok := meta.ContextMeta(ctx)
		if ok {
			meta.Service = "alive"
			meta.Method = "alive"
		}
	}
}
