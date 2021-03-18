package logging

import (
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"time"

	log "github.com/inconshreveable/log15"
)

func InitFileLogger(logger log.Logger, logPath string) error {
	format := "2006-01-02.json"
	err := os.MkdirAll(logPath, os.ModePerm)
	if err != nil {
		return err
	}
	// set default logger
	logHandler, err := log.FileHandler(logPath+(time.Now().Format(format)), log.JsonFormat())
	if err != nil {
		return err
	}
	logger.SetHandler(
		log.MultiHandler(
			logHandler,
			log.StreamHandler(os.Stdout, log.TerminalFormat()), // add a readable one for the terminal
		))

	// delete old log files
	logFiles, err := filepath.Glob(logPath + "*.json")
	if err != nil {
		return err
	}
	for _, file := range logFiles {
		date, _ := time.Parse(logPath+format, file)
		if int(time.Since(date).Hours()/24) > 7 {
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
