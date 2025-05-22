package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/ardanlabs/conf/v3"
	"github.com/goccy/go-yaml"
	"github.com/top-solution/go-libs/version/v2"
)

var baseFlags = struct {
	// The config file path: it's only read from flags or env vars
	ConfigFile string `yaml:"-" conf:"default:conf.yml"`
	// The build info of the app It's used by ardanlabs/conf
	Build string `yaml:"-"`
	// The description of the app version. It's used by ardanlabs/conf
	Desc string `yaml:"-"`
}{}

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
		slog.Warn("unable to read YAML config from " + path + ": skipping")
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
	content, err := os.ReadFile(path)
	if err != nil {
		return c, fmt.Errorf("read config yaml: %w", err)
	}
	return yamlConfParser{
		data: content,
	}, nil
}
