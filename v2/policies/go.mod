module github.com/top-solution/go-libs/v2/policies

go 1.24.1

replace github.com/top-solution/go-libs/v2/keys => ../keys

replace github.com/top-solution/go-libs/v2/authorizer => ../authorizer

require (
	github.com/ory/ladon v1.3.0
	github.com/top-solution/go-libs/v2/authorizer v0.0.0-00010101000000-000000000000
	github.com/top-solution/go-libs/v2/keys v0.0.0-00010101000000-000000000000
)

require (
	github.com/dlclark/regexp2 v1.2.0 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.2 // indirect
	github.com/hashicorp/golang-lru v0.5.0 // indirect
	github.com/ory/pagination v0.0.1 // indirect
	github.com/pkg/errors v0.8.0 // indirect
)
