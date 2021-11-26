package logging

import (
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"time"

	log "github.com/inconshreveable/log15"
	"gitlab.com/top-solution/go-libs/frequency"
)

// LogConfig contains the log confgiuration
type LogConfig struct {
	Path       string `yaml:"path"`
	Expiration struct {
		Frequency frequency.Frequency `yaml:"frequency"`
	} `yaml:"expiration"`
}

func InitFileLogger(logger log.Logger, config LogConfig) error {
	if config.Path == "" {
		config.Path = "log"
	}
	if config.Expiration.Frequency.IsZero() {
		config.Expiration.Frequency, _ = frequency.ParseFrequency("1w")
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
			logHandler,
			log.StreamHandler(os.Stdout, log.TerminalFormat()), // add a readable one for the terminal
		))

	// delete old log files
	logFiles, err := filepath.Glob(filepath.Join(config.Path, "*.json"))
	if err != nil {
		return err
	}
	for _, file := range logFiles {
		date, _ := time.Parse(filepath.Join(config.Path, format), file)
		if config.Expiration.Frequency.ShouldRun(date, time.Now()) {
			log.Debug("Deleting old log file:"+file, "age", time.Since(date))
			err := os.Remove(file)
			if err != nil {
				log.Error(err.Error())
			}
		}
	}

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
