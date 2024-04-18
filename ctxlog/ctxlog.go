// TODO: migrate to log/slog
package ctxlog

import (
	"context"

	log "github.com/inconshreveable/log15"
)

type logKey string

// LogKey is the key used to store and retrieve the entry log
// You'll probably want to store the initial entry at the very beginning of the request/trace/whatever
var LogKey = logKey("ctxlog")

var FieldsKey = logKey("ctxlog_fields")

// WithField adds a single field to the entry saved in the given context.
// If there is no entry saved in the context, the default one will be created
func WithField(ctx context.Context, key string, value interface{}) context.Context {
	fields := getFields(ctx)

	fields[key] = value

	return context.WithValue(ctx, FieldsKey, fields)
}

// WithFields adds a map of fields to the entry saved in the given context.
// If there is no entry saved in the context, the default one will be created
func WithFields(ctx context.Context, fields map[string]interface{}) context.Context {
	f := getFields(ctx)

	for k, v := range fields {
		f[k] = v
	}

	return context.WithValue(ctx, FieldsKey, f)
}

// Debug will call the Debug function on the entry saved in the given context.
// If there is no entry saved in the context, the default one will be created
func Debug(ctx context.Context, msg string, args ...interface{}) {
	entry := getEntry(ctx)

	entry.Debug(msg, getArgs(ctx, args...)...)
}

// Info will call the Info function on the entry saved in the given context.
// If there is no entry saved in the context, the default one will be created.
// See the Debug Example
func Info(ctx context.Context, msg string, args ...interface{}) {
	entry := getEntry(ctx)

	entry.Info(msg, getArgs(ctx, args...)...)
}

// Warn will call the Warn function on the entry saved in the given context.
// If there is no entry saved in the context, the default one will be created.
// See the Debug Example
func Warn(ctx context.Context, msg string, args ...interface{}) {
	entry := getEntry(ctx)

	entry.Warn(msg, getArgs(ctx, args...)...)
}

// Error will call the Warn function on the entry saved in the given context.
// If there is no entry saved in the context, the default one will be created.
// See the Debug Example
func Error(ctx context.Context, msg string, args ...interface{}) {
	entry := getEntry(ctx)

	entry.Error(msg, getArgs(ctx, args...)...)
}

func getEntry(ctx context.Context) log.Logger {
	entry, ok := ctx.Value(LogKey).(log.Logger)
	if !ok {
		entry = log.Root()
	}

	return entry
}

func getFields(ctx context.Context) map[string]interface{} {
	fields, ok := ctx.Value(FieldsKey).(map[string]interface{})
	if !ok {
		fields = make(map[string]interface{})
	}

	return fields
}

func getArgs(ctx context.Context, args ...interface{}) []interface{} {
	var ctxs []interface{}
	fields := getFields(ctx)
	for k, v := range fields {
		ctxs = append(ctxs, k, v)
	}

	return append(ctxs, args...)
}
