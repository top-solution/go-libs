package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/dimfeld/httptreemux/v5"
	goahttp "goa.design/goa/v3/http"
)

type (
	// Muxer is the HTTP request multiplexer interface used by the generated
	// code. ServerHTTP must match the HTTP method and URL of each incoming
	// request against the list of registered patterns and call the handler
	// for the corresponding method and the pattern that most closely
	// matches the URL.
	//
	// The patterns may include wildcards that identify URL segments that
	// must be captured.
	//
	// There are two forms of wildcards the implementation must support:
	//
	//   - "{name}" wildcards capture a single path segment, for example the
	//     pattern "/images/{name}" captures "/images/favicon.ico" and adds
	//     the key "name" with the value "favicon.ico" to the map returned
	//     by Vars.
	//
	//   - "{*name}" wildcards must appear at the end of the pattern and
	//     captures the entire path starting where the wildcard matches. For
	//     example the pattern "/images/{*filename}" captures
	//     "/images/public/thumbnail.jpg" and associates the key key
	//     "filename" with "public/thumbnail.jpg" in the map returned by
	//     Vars.
	//
	// The names of wildcards must match the regular expression
	// "[a-zA-Z0-9_]+".
	Muxer interface {
		// Handle registers the handler function for the given method
		// and pattern.
		Handle(method, pattern string, handler http.HandlerFunc)

		// ServeHTTP dispatches the request to the handler whose method
		// matches the request method and whose pattern most closely
		// matches the request URL.
		ServeHTTP(http.ResponseWriter, *http.Request)

		// Vars returns the path variables captured for the given
		// request.
		Vars(*http.Request) map[string]string
	}

	// Mux is the default Muxer implementation. It leverages the
	// httptreemux router and simply substitutes the syntax used to define
	// wildcards from ":wildcard" and "*wildcard" to "{wildcard}" and
	// "{*wildcard}" respectively.
	Mux struct {
		*httptreemux.ContextMux
	}
)

// NewMuxer returns a Muxer implementation based on the httptreemux router.
func NewMuxer() *Mux {
	r := httptreemux.NewContextMux()
	r.EscapeAddedRoutes = true
	r.RedirectBehavior = httptreemux.UseHandler
	r.NotFoundHandler = func(w http.ResponseWriter, req *http.Request) {
		ctx := context.WithValue(req.Context(), goahttp.AcceptTypeKey, req.Header.Get("Accept"))
		enc := goahttp.ResponseEncoder(ctx, w)
		w.WriteHeader(http.StatusNotFound)
		_ = enc.Encode(goahttp.NewErrorResponse(ctx, fmt.Errorf("404 page not found")))
	}
	return &Mux{r}
}

// Handle maps the wildcard format used by goa to the one used by httptreemux.
func (m *Mux) Handle(method, pattern string, handler http.HandlerFunc) {
	m.ContextMux.Handle(method, treemuxify(pattern), handler)
}

// Vars extracts the path variables from the request context.
func (m *Mux) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	// IIS randomly unescapes some characters, try to recover the original URL
	if r.Header.Get("X-Unencoded-Url") != "" {
		r.RequestURI = r.Header.Get("X-Unencoded-Url")
	}
	m.ContextMux.ServeHTTP(rw, r)
}

// Vars extracts the path variables from the request context.
func (m *Mux) Vars(r *http.Request) map[string]string {
	return httptreemux.ContextParams(r.Context())
}

var wildSeg = regexp.MustCompile(`/{([a-zA-Z0-9_]+)}`)
var wildPath = regexp.MustCompile(`/{\*([a-zA-Z0-9_]+)}`)

func treemuxify(pattern string) string {
	pattern = wildSeg.ReplaceAllString(pattern, "/:$1")
	pattern = wildPath.ReplaceAllString(pattern, "/*$1")
	return pattern
}
