package logging

import (
	"context"
	"log/slog"
)

type MultiHandler []slog.Handler

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

func (m MultiHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, h := range m {
		err := h.Handle(ctx, record)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return nil
}

func (m MultiHandler) WithGroup(name string) slog.Handler {
	return nil
}
