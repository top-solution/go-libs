package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/ardanlabs/conf/v2"
	confYaml "github.com/ardanlabs/conf/v2/yaml"
	"gitlab.com/top-solution/go-libs/frequency"
	"gitlab.com/top-solution/go-libs/version"
)

var baseFlags = struct {
	// The config file path: it's only read from flags or env vars
	ConfigFile string `yaml:"-" conf:"default:conf.yml"`
	// The build info of the app
	Build string `yaml:"-"`
	// The
	Desc string `yaml:"-"`
}{}

// ParseConfig parses a file config, fetching the (optional) config file name from a flag or env var
func ParseConfig(cfg interface{}) error {
	// Build version info
	versionInfo := version.GetInfo()
	baseFlags.Desc = fmt.Sprintf("%s - %s", versionInfo.Commit, versionInfo.BuildDate)
	baseFlags.Build = versionInfo.Version

	// Parse base flags
	help, err := conf.Parse("", &baseFlags)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			if !strings.HasPrefix(help, "Version:") {
				help, _ = conf.UsageInfo("", cfg)
			}
			fmt.Println(help)
			os.Exit(0)
			return nil
		}
		return fmt.Errorf("parsing config: %w", err)
	}
	// Parse config file
	return ParseConfigWithPrefix(cfg, baseFlags.ConfigFile, "")
}

// ParseConfig parses a file config, given the file path, expecting the prefix
// See ardanlabs/conf's docs to see the usefulness of a prefix
func ParseConfigWithPrefix(cfg interface{}, path string, prefix string) error {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	help, err := conf.Parse(prefix, cfg, confYaml.WithData(content))
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
			os.Exit(0)
			return nil
		}
		return fmt.Errorf("parsing config: %w", err)
	}
	return nil
}

// EmailConfig LogConfig is a default config struct for e-mail
type EmailConfig struct {
	Host              string `yaml:"host"`
	Password          string `yaml:"password"`
	Sender            string `yaml:"sender"`
	Port              int    `yaml:"smtp_port"`
	User              string `yaml:"smtp_user"`
	TemplateDirectory string `yaml:"template_directory"`
}

// LogConfig  is a default config struct  for logs
type LogConfig struct {
	Path       string `yaml:"path" conf:"default:logs"`
	Expiration struct {
		Frequency frequency.Frequency `yaml:"frequency"`
	} `yaml:"expiration"`
}

// DBConfig is a default config struct used to connect to a database
type DBConfig struct {
	// Driver contains the driver name
	Driver string `yaml:"driver"`
	// Type contains the DB type: it's a MSSQL thing
	Type string `yaml:"type" conf:"default:sqlserver"`
	// Server contains the db host address
	Server string `yaml:"server"`
	// Port contains the db port
	Port int `yaml:"port"`
	// User contaisn the user to access the db
	User string `yaml:"user"`
	// Password contains the password to access the db
	Password string `yaml:"password"`
	// DB contains the DB name
	DB string `yaml:"db"`
	// MigrationsPath contains the path for the migration sql files
	MigrationsPath string `yaml:"migrations_path" conf:"default:sql"`
}
