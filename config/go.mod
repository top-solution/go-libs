module github.com/top-solution/go-libs/config

go 1.22.2

replace github.com/top-solution/go-libs/version => ../version

require (
	github.com/ardanlabs/conf/v3 v3.1.7
	github.com/goccy/go-yaml v1.11.3
	github.com/inconshreveable/log15 v2.16.0+incompatible
	github.com/serjlee/frequency v1.1.0
	github.com/top-solution/go-libs/version v0.18.5
)

require (
	github.com/fatih/color v1.10.0 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/term v0.19.0 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
)
