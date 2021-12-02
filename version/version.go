package version

import (
	"fmt"
)

type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
}

var (
	commit    = "missing"
	version   = "missing"
	buildDate = "missing"
)

func GetInfo() Info {
	return Info{
		BuildDate: buildDate,
		Commit:    commit,
		Version:   version,
	}
}

func Print() string {
	infoString := `
version   : %s
commit    : %s
build date: %s
`
	return fmt.Sprintf(infoString, version, commit, buildDate)
}

func SetVersion(v string) {
	version = v
}

func GetVersion() string {
	return version
}
