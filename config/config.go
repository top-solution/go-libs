package config

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/ardanlabs/conf/v2"
	confYaml "github.com/ardanlabs/conf/v2/yaml"
)

func ParseConfig(cfg interface{}, path string, fsys fs.FS) error {
	return ParseConfigWithPrefix(cfg, path, fsys, "")
}

func ParseConfigWithPrefix(cfg interface{}, path string, fsys fs.FS, prefix string) error {
	content, err := fs.ReadFile(fsys, path)
	if err != nil {
		return err
	}
	help, err := conf.Parse(prefix, cfg, confYaml.WithData(content))
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
			return nil
		}
		return fmt.Errorf("parsing config: %w", err)
	}
	return nil
}
