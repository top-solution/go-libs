module github.com/top-solution/go-libs/middlewares

go 1.22.2

replace github.com/top-solution/go-libs/config => ../config

replace github.com/top-solution/go-libs/scheduler => ../scheduler

replace github.com/top-solution/go-libs/version => ../version

replace github.com/top-solution/go-libs/ctxlog => ../ctxlog

replace github.com/top-solution/go-libs/fs => ../fs

require (
	github.com/dimfeld/httptreemux/v5 v5.5.0
	github.com/inconshreveable/log15 v2.16.0+incompatible
	github.com/top-solution/go-libs/ctxlog v0.0.0-00010101000000-000000000000
	github.com/top-solution/go-libs/fs v0.0.0-00010101000000-000000000000
	goa.design/goa/v3 v3.16.1
	goa.design/plugins v2.2.6+incompatible
)

require (
	github.com/dimfeld/httppath v0.0.0-20170720192232-ee938bf73598 // indirect
	github.com/go-chi/chi/v5 v5.0.12 // indirect
	github.com/go-openapi/loads v0.22.0 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/google/gxui v0.0.0-20151028112939-f85e0a97b3a4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.1 // indirect
	github.com/manveru/faker v0.0.0-20171103152722-9fbc68a78c4d // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/sergi/go-diff v1.3.1 // indirect
	github.com/smartystreets/goconvey v1.8.1 // indirect
	github.com/zach-klippenstein/goregen v0.0.0-20160303162051-795b5e3961ea // indirect
	goa.design/goa v2.2.5+incompatible // indirect
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/net v0.24.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/term v0.19.0 // indirect
	golang.org/x/tools v0.20.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
