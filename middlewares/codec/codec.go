package codec

import (
	"context"
	"encoding/csv"
	"io"
	"net/http"

	goahttp "goa.design/goa/v3/http"
)

// ResponseEncoder is a custom goa Encoder able to output data in CSV format
// It can support any endpoint returning a list of fields, a map of fields or struct with a "Data" field containing a  valid list or map (ie: pagination structs)
// Note it doesn't properly handle nested structs.
//
// Clients are supposed to set the Accept header to application/csv in order to get the actual CSV
func ResponseEncoder(ctx context.Context, w http.ResponseWriter) goahttp.Encoder {
	var accept string
	if a := ctx.Value(goahttp.AcceptTypeKey); a != nil {
		accept = a.(string)
	}
	if accept == "application/csv" {
		goahttp.SetContentType(w, accept)
		return newCSVEncoder(w)
	}
	return goahttp.ResponseEncoder(ctx, w)
}

func newCSVEncoder(w io.Writer) goahttp.Encoder {
	wr := csv.NewWriter(w)
	wr.Comma = ';'
	return &CSVEncoder{wr}
}
