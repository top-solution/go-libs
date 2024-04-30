module github.com/top-solution/go-libs/policies

go 1.22.2

replace github.com/top-solution/go-libs/authorizer => ../authorizer

replace github.com/top-solution/go-libs/keys => ../keys

replace github.com/top-solution/go-libs/ctxlog => ../ctxlog

require (
	github.com/ory/ladon v1.3.0
	github.com/top-solution/go-libs/authorizer v0.0.0-00010101000000-000000000000
	github.com/top-solution/go-libs/keys v0.0.0-00010101000000-000000000000
	goa.design/goa/v3 v3.16.1
)

require (
	github.com/dlclark/regexp2 v1.2.0 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/golang-lru v0.5.0 // indirect
	github.com/inconshreveable/log15 v2.16.0+incompatible // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ory/pagination v0.0.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/top-solution/go-libs/ctxlog v0.18.5 // indirect
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/term v0.19.0 // indirect
)
