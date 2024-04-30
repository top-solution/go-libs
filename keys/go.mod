module github.com/top-solution/go-libs/keys

go 1.22.2

replace github.com/top-solution/go-libs/ctxlog => ../ctxlog

require (
	github.com/golang-jwt/jwt/v4 v4.5.0
	github.com/top-solution/go-libs/ctxlog v0.18.5
)

require (
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/inconshreveable/log15 v2.16.0+incompatible // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/term v0.19.0 // indirect
)
