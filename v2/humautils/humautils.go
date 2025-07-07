package humautils

import (
	"context"
	"log/slog"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const RESOURCE = "resource"

type BodyResponse[T any] struct {
	Body T
}

// Body is a convenience function to create a BodyResponse with the given body. It only works for comparable types to make sure we use List when we want to return a list.
func Body[T comparable](body T) *BodyResponse[T] {
	return &BodyResponse[T]{
		Body: body,
	}
}

// List is the same as Body, but for slices: it initializes the slice with an empty slice to avoid JSON marshaling issues with nil ones.
func List[T any](body []T) *BodyResponse[[]T] {
	return &BodyResponse[[]T]{
		Body: append(make([]T, 0), body...), // Initialize with an empty slice, so we do't have to fiddle with JSON marshaling of nil slices
	}
}

// Map is the same as List, but for maps.
func Map[T any](body map[string]T) *BodyResponse[map[string]T] {
	if body == nil {
		body = make(map[string]T)
	}
	return &BodyResponse[map[string]T]{
		Body: body,
	}
}

func RegisterEndpoint[I, O any](api huma.API, resource string, op huma.Operation, handler func(context.Context, *I) (*O, error)) {
	if op.Metadata == nil {
		op.Metadata = map[string]any{}
	}
	op.Metadata[RESOURCE] = resource
	op.Tags = append(op.Tags, kebabToTitleCase(resource))
	if op.Summary == "" {
		op.Summary = kebabToTitleCase(op.OperationID)
	}

	huma.Register(api, op, func(ctx context.Context, input *I) (*O, error) {
		output, err := handler(ctx, input)
		if err != nil {
			if _, ok := err.(huma.StatusError); !ok {
				slog.Error("Unexpected error",
					"status", 500,
					"path", op.Path,
					"method", op.Method,
					"error", err,
				)
			}
		}

		return output, err
	})
}

func kebabToTitleCase(input string) string {
	withSpaces := strings.ReplaceAll(input, "-", " ")
	caser := cases.Title(language.English)
	return caser.String(withSpaces)
}
