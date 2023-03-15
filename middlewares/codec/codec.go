package codec

import (
	"context"
	"encoding/csv"
	"io"
	"net/http"

	goahttp "goa.design/goa/v3/http"
)

// ResponseEncoder is a custom goa Encoder able to output data in CSV format
// It can support any endpoint returning:
// - A slice of structs
// - A struct containing a "Data" field which is a slice of structs (ie: Pagination wrappers)
// - A simple map
// - A struct containing a "Data" field which is a simple map
//
// Note: nested structs are handled only for the first level, and only for slices
//
// Clients are supposed to set the Accept header to application/csv in order to get the actual CSV
func ResponseEncoder(ctx context.Context, w http.ResponseWriter) goahttp.Encoder {
	var ct string
	// Read Accept header from client
	if a := ctx.Value(goahttp.AcceptTypeKey); a != nil {
		ct = a.(string)
	}

	// Read ContentType("") DSL from goa design (for enforced CSV endpoints)
	if a := ctx.Value(goahttp.ContentTypeKey); a != nil {
		ct = a.(string)
	}

	if ct == "application/csv" {
		goahttp.SetContentType(w, ct)
		return newCSVEncoder(w)
	}
	return goahttp.ResponseEncoder(ctx, w)
}

func newCSVEncoder(w io.Writer) goahttp.Encoder {
	wr := csv.NewWriter(w)
	wr.Comma = ';'
	return &CSVEncoder{wr}
}
