package ctxlog

import (
	"context"

	log "github.com/inconshreveable/log15"
)

type logKey string

// LogKey is the key used to store and retrieve the entry log
// You'll probably want to store the initial entry at the very beginning of the request/trace/whatever
var LogKey = logKey("ctxlog")

// WithField adds a single field to the entry saved in the given context.
// If there is no entry saved in the context, the default one will be created
func WithField(ctx context.Context, key string, value interface{}) context.Context {
	entry := getEntry(ctx)

	entry = entry.New(key, value)

	return context.WithValue(ctx, LogKey, entry)
}

// WithFields adds a map of fields to the entry saved in the given context.
// If there is no entry saved in the context, the default one will be created
func WithFields(ctx context.Context, fields ...interface{}) context.Context {
	entry := getEntry(ctx)

	entry = entry.New(fields...)

	return context.WithValue(ctx, LogKey, entry)
}

// Debug will call the Debug function on the entry saved in the given context.
// If there is no entry saved in the context, the default one will be created
func Debug(ctx context.Context, args ...interface{}) {
	entry := getEntry(ctx)

	entry.Debug("", args...)
}

// Info will call the Info function on the entry saved in the given context.
// If there is no entry saved in the context, the default one will be created.
// See the Debug Category
func Info(ctx context.Context, args ...interface{}) {
	entry := getEntry(ctx)

	entry.Info("", args...)
}

// Warn will call the Warn function on the entry saved in the given context.
// If there is no entry saved in the context, the default one will be created.
// See the Debug Category
func Warn(ctx context.Context, args ...interface{}) {
	entry := getEntry(ctx)

	entry.Warn("", args...)
}

// Error will call the Warn function on the entry saved in the given context.
// If there is no entry saved in the context, the default one will be created.
// See the Debug Category
func Error(ctx context.Context, args ...interface{}) {
	entry := getEntry(ctx)

	entry.Error("", args...)
}

func getEntry(ctx context.Context) log.Logger {
	entry, ok := ctx.Value(LogKey).(log.Logger)
	if !ok {
		entry = log.Root()
	}

	return entry
}
