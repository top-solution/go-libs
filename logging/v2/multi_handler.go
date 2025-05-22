package logging

import (
	"context"
	"log/slog"
)

// MultiHandler is a handler that can be used to log to multiple handlers at once
type MultiHandler []slog.Handler

// Enabled returns true if at least one of the handlers is enabled
func (m MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if len(m) == 0 {
		return false
	}
	for _, h := range m {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle logs the record to all handlers
func (m MultiHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, h := range m {
		err := h.Handle(ctx, record)
		if err != nil {
			return err
		}
	}
	return nil
}

// WithAttrs returns a new MultiHandler with the given attributes
func (m MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	multiHandlerAttrs := MultiHandler{}
	for _, h := range m {
		multiHandlerAttrs = append(multiHandlerAttrs, h.WithAttrs(attrs))
	}
	return multiHandlerAttrs
}

// WithGroup returns a new MultiHandler with the given group
func (m MultiHandler) WithGroup(name string) slog.Handler {
	multiHandlerGroup := MultiHandler{}
	for _, h := range m {
		multiHandlerGroup = append(multiHandlerGroup, h.WithGroup(name))
	}
	return multiHandlerGroup
}
