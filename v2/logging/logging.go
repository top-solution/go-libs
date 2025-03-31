package logging

import (
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
)

// LogConfig  is a default config struct  for logs
type LogConfig struct {
	Path       string              `yaml:"path" conf:"default:logs/,help:The directory to use to store logs"`
	Expiration frequency.Frequency `yaml:"expiration" conf:"default:1w,help:How long should the logs kept"`
	Level      string              `yaml:"level" conf:"default:debug,help:The log level (can be: debug, info, warn, error, crit)"`
	Format     string              `yaml:"format" conf:"default:terminal,help:The log format (can be: terminal, json)"`
}

// FilterLogLevel wraps a log handler with a log level filter, given the log config
func FilterLogLevel(config LogConfig) slog.Level {
	logLevel, _ := SlogLvlFromString(config.Level)

	return logLevel

}

// GetSlogHandlerByFormat returns the log handler given the log config
func GetSlogHandlerByFormat(config LogConfig) slog.Handler {
	if config.Format == "json" {
		return slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: FilterLogLevel(config),
		})
	}
	return tint.NewHandler(os.Stdout, &tint.Options{
		Level:      FilterLogLevel(config),
		TimeFormat: time.DateTime,
	})
}

// InitTerminalLogger sets the default logger to only print in the terminal
func InitTerminalLogger(config LogConfig) {
	logger := slog.New(
		GetSlogHandlerByFormat(config),
	)
	slog.SetDefault(logger)
}

// InitFileLogger sets up the default logger to both print in the terminal and in a JSON logfile
func InitFileLogger(config LogConfig) error {
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

	jsonSlogHandlerForFile := slog.NewJSONHandler(
		file,
		&slog.HandlerOptions{
			Level: FilterLogLevel(config),
		},
	)

	slogHandlers := MultiHandler{jsonSlogHandlerForFile, GetSlogHandlerByFormat(config)}

	slog.SetDefault(slog.New(slogHandlers))

	// Check and delete old logs hourly, starting now
	go func() {
		for {
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
			<-time.After(time.Hour)
		}
	}()

	return nil
}

type goaServer interface {
	Service() string
	Use(m func(http.Handler) http.Handler)
}

// LogGoaEndpoints is a small helper that logs all the endpoints of a goa server
func LogGoaEndpoints(srv goaServer) {
	r := reflect.ValueOf(srv)
	mounts := reflect.Indirect(r).FieldByName("Mounts")

	for i := 0; i < mounts.Len(); i++ {
		m := reflect.Indirect(mounts.Index(i))
		slog.Info("mounted", "svc", srv.Service(), "method", m.FieldByName("Method"), "verb", m.FieldByName("Verb"), "pattern", m.FieldByName("Pattern"))
	}
}

// SlogLvlFromString converts a string to a slog level
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
