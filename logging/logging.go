package logging

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"github.com/serjlee/frequency"
	"github.com/top-solution/go-libs/config"
	"github.com/top-solution/go-libs/scheduler"
)

var cleanupLogsTask *scheduler.Entry

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

// FilterLogLevel wraps a log handler with a log level filter, given the log config
func FilterLogLevel(config config.LogConfig) slog.Level {
	logLevel, _ := SlogLvlFromString(config.Level)

	return logLevel

}

// InitTerminalLogger sets up a logger (ie: log.Root()) to only print in the terminal
func InitTerminalLogger(config config.LogConfig) {
	logger := slog.New(
		tint.NewHandler(os.Stdout, &tint.Options{
			Level:      FilterLogLevel(config),
			TimeFormat: time.Kitchen,
		}),
	)
	slog.SetDefault(logger)
}

// InitFileLogger sets up a logger (ie: log.Root()) to both print in the terminal and in a JSON logfile
func InitFileLogger(config config.LogConfig) error {
	if config.Path == "" {
		config.Path = "log"
	}
	if config.Expiration.IsZero() {
		config.Expiration, _ = frequency.ParseFrequency("1w")
	}

	format := "2006-01-02 15-04-05.json"
	err := os.MkdirAll(config.Path, os.ModePerm)
	if err != nil {
		return err
	}
	// set default logger
	file, err := os.OpenFile(filepath.Join(config.Path, time.Now().Format(format)), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}

	//w := io.MultiWriter(file, os.Stdout)

	jsonSlogHandler := slog.NewJSONHandler(
		file,
		&slog.HandlerOptions{
			Level: FilterLogLevel(config),
		},
	)

	terminalSlogHandler := tint.NewHandler(os.Stdout, &tint.Options{
		Level:      FilterLogLevel(config),
		TimeFormat: time.Kitchen,
	})

	slogHandlers := MultiHandler{jsonSlogHandler, terminalSlogHandler}

	slog.SetDefault(slog.New(slogHandlers))

	// cleanup old logs
	cleanupFn := func() {
		logFiles, err := filepath.Glob(filepath.Join(config.Path, "*.json"))
		if err != nil {
			slog.Error(err.Error())
			return
		}
		for _, file := range logFiles {
			date, _ := time.Parse(filepath.Join(config.Path, format), file)

			if config.Expiration.ShouldRun(date, time.Now()) {
				slog.Debug("Deleting old log file:"+file, "age", time.Since(date))
				err := os.Remove(file)
				if err != nil {
					slog.Error(err.Error())
				}
			}
		}
	}
	cleanupFn()

	// InitFileLogger was called twice: weird, but we can handle it
	if cleanupLogsTask != nil {
		cleanupLogsTask.TaskFn = cleanupFn
	}
	// Check and delete old logs hourly
	cleanupLogsTask = scheduler.DefaultScheduler.Every(frequency.FromDuration(1 * time.Hour)).Do(cleanupFn)

	return nil
}

type GoaServer interface {
	Service() string
	Use(m func(http.Handler) http.Handler)
}

func LogGoaEndpoints(srv GoaServer) {
	r := reflect.ValueOf(srv)
	mounts := reflect.Indirect(r).FieldByName("Mounts")

	for i := 0; i < mounts.Len(); i++ {
		m := reflect.Indirect(mounts.Index(i))
		slog.Info("mounted", "svc", srv.Service(), "method", m.FieldByName("Method"), "verb", m.FieldByName("Verb"), "pattern", m.FieldByName("Pattern"))
	}
}

func SlogLvlFromString(lvlString string) (slog.Level, error) {
	switch lvlString {
	case "debug", "dbug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error", "eror":
		return slog.LevelError, nil
	default:
		// try to catch e.g. "INFO", "WARN" without slowing down the fast path
		lower := strings.ToLower(lvlString)
		if lower != lvlString {
			return SlogLvlFromString(lower)
		}
		return slog.LevelDebug, fmt.Errorf("slog: unknown level: %v", lvlString)
	}
}
