package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/ardanlabs/conf/v2"
	"github.com/goccy/go-yaml"
	log "github.com/inconshreveable/log15"
	"github.com/serjlee/frequency"
	"github.com/top-solution/go-libs/version"
)

var baseFlags = struct {
	// The config file path: it's only read from flags or env vars
	ConfigFile string `yaml:"-" conf:"default:conf.yml"`
	// The build info of the app It's used by ardanlabs/conf
	Build string `yaml:"-"`
	// The description of the app version. It's used by ardanlabs/conf
	Desc string `yaml:"-"`
}{}

type yamlConfParser struct {
	data []byte
}

func (y yamlConfParser) Process(prefix string, cfg interface{}) error {
	err := yaml.Unmarshal(y.data, cfg)
	if err != nil {
		return fmt.Errorf("unmarshal yaml: %w", err)
	}
	return nil
}

func readYamlConf(path string) (c yamlConfParser, err error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return c, fmt.Errorf("read config yaml: %w", err)
	}
	return yamlConfParser{
		data: content,
	}, nil
}

// ParseConfig parses a file config, fetching the (optional) config file name from a flag or env var
// It expects a config file passed via the --config-file flag or CONFIG_FILE env var, defaulting to conf.yml
// It will also provide a way to set config values via env vars or flags, and add a --version and --help command in the final executable
//
// The --help text will be populated by analyzing the struct of the passed cfg
// The --version text will be populated via github.com/top-solution/go-libs/version
func ParseConfigAndVersion(cfg interface{}) error {
	// Build version info
	versionInfo := version.GetInfo()
	baseFlags.Desc = fmt.Sprintf("Commit %s built @ %s", versionInfo.Commit, versionInfo.BuildDate)
	baseFlags.Build = versionInfo.Version

	// Parse base flags
	help, err := conf.Parse("", &baseFlags)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			helpActuallyWanted := !strings.HasPrefix(help, "Version:")
			if helpActuallyWanted {
				help, _ = conf.UsageInfo("", cfg)
			}
			fmt.Println(help)
			if helpActuallyWanted {
				fmt.Println("BASE OPTIONS")
				fmt.Println("  --config-file\t\t <string> set the config file path (default: conf.yml)")
			}
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
	var help string
	var err error
	yamlConfData, err := readYamlConf(path)
	if err == nil {
		help, err = conf.Parse(prefix, cfg, yamlConfData)
	} else {
		log.Root().Warn("unable to read YAML config from " + path + ": skipping")
		help, err = conf.Parse(prefix, cfg)
	}
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
	Path       string              `yaml:"path" conf:"default:logs/,help:The directory to use to store logs"`
	Expiration frequency.Frequency `yaml:"expiration" conf:"default:1w,help:How long should the logs kept"`
}

// DBConfig is a default config struct used to connect to a database
type DBConfig struct {
	// Driver is the driver name
	Driver string `yaml:"driver" conf:"help:The db driver name"`
	// Server is db host address
	Server string `yaml:"server" conf:"help:The db host"`
	// Port is the db port
	Port int `yaml:"port" conf:"help:The db name"`
	// User is the db user
	User string `yaml:"user" conf:"help:The db user"`
	// Password is the password for the db user
	Password string `yaml:"password" conf:"help:The db user password"`
	// DB is the db name
	DB         string `yaml:"db" conf:"help:The name of the DB"`
	Migrations struct {
		Run  bool   `yaml:"run" conf:"default:false,help:If true, migrations will be run on app startup"`
		Path string `yaml:"path" conf:"default:sql,help:The path to the directory containing the Goose-compatible SQL migrations"`
	} `yaml:"migrations"`
}
