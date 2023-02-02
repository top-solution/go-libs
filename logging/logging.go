package logging

import (
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"time"

	log "github.com/inconshreveable/log15"
	"github.com/serjlee/frequency"
	"github.com/top-solution/go-libs/config"
	"github.com/top-solution/go-libs/scheduler"
)

var cleanupLogsTask *scheduler.Entry

// FilterLogLevel wraps a log handler with a log level filter, given the log config
func FilterLogLevel(logHandler log.Handler, config config.LogConfig) log.Handler {
	lvl, _ := log.LvlFromString(config.Level)

	return log.LvlFilterHandler(
		lvl,
		logHandler)
}

func InitFileLogger(logger log.Logger, config config.LogConfig) error {
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
	logHandler, err := log.FileHandler(filepath.Join(config.Path, time.Now().Format(format)), log.JsonFormat())
	if err != nil {
		return err
	}
	logger.SetHandler(
		log.MultiHandler(
			FilterLogLevel(logHandler, config),
			FilterLogLevel(log.StreamHandler(os.Stdout, log.TerminalFormat()), config), // add a readable one for the terminal
		))

	// cleanup old logs
	cleanupFn := func() {
		logFiles, err := filepath.Glob(filepath.Join(config.Path, "*.json"))
		if err != nil {
			log.Error(err.Error())
			return
		}
		for _, file := range logFiles {
			date, _ := time.Parse(filepath.Join(config.Path, format), file)

			if config.Expiration.ShouldRun(date, time.Now()) {
				log.Debug("Deleting old log file:"+file, "age", time.Since(date))
				err := os.Remove(file)
				if err != nil {
					log.Error(err.Error())
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
		log.Info("mounted", "svc", srv.Service(), "method", m.FieldByName("Method"), "verb", m.FieldByName("Verb"), "pattern", m.FieldByName("Pattern"))
	}
}
