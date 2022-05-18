package middlewares

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	log "github.com/inconshreveable/log15"
	"github.com/top-solution/go-libs/middlewares/meta"
	goahttp "goa.design/goa/v3/http"
	goa "goa.design/goa/v3/pkg"
)

type Server struct {
	Log log.Logger
	Dec func(r *http.Request) goahttp.Decoder
	Enc func(context.Context, http.ResponseWriter) goahttp.Encoder
	Mux goahttp.Muxer
	Eh  func(context.Context, http.ResponseWriter, error)
}

func New(entry log.Logger) Server {
	dec := goahttp.RequestDecoder
	enc := goahttp.ResponseEncoder

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
