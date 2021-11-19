package logging

import (
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	log "github.com/inconshreveable/log15"
)

// LogConfig contains the log confgiuration
type LogConfig struct {
	Path             string `yaml:"path"`
	ExpirationInDays int    `yaml:"expiration_in_days"`
}

func InitFileLogger(logger log.Logger, config LogConfig) error {
	if config.Path == "" {
		config.Path = "log"
	}
	if config.ExpirationInDays == 0 {
		config.ExpirationInDays = 7
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
		if int(time.Since(date).Hours()/24) > config.ExpirationInDays {
			log.Debug("Deleting old log file:"+file, "age", time.Since(date))
			err := os.Remove(file)
			if err != nil {
				log.Error(err.Error())
			}
		}
	}

	return nil
}

func expireDate(expireTime time.Time, expire string) (*time.Time, error) {
	splitted := strings.Split(expire, " ")
	data := []int{}
	for _, elem := range splitted {
		intTmp, err := strconv.Atoi(elem)
		if err != nil {
			return nil, err
		}
		data = append(data, intTmp)
	}

	expireTime.AddDate(data[3], data[4], data[5])
	expireTime.Add(time.Second * time.Duration(data[0]))
	expireTime.Add(time.Minute * time.Duration(data[1]))
	expireTime.Add(time.Hour * time.Duration(data[2]))

	return &expireTime, nil
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
