module github.com/top-solution/go-libs/bobops/v2

go 1.24.1

replace github.com/top-solution/go-libs/dbutils/v2 => ../../dbutils/v2

require (
	github.com/lib/pq v1.10.9
	github.com/stephenafamo/bob v0.34.2
	github.com/top-solution/go-libs/dbutils/v2 v2.0.0-00010101000000-000000000000
)

require (
	github.com/aarondl/json v0.0.0-20221020222930-8b0db17ef1bf // indirect
	github.com/aarondl/opt v0.0.0-20230114172057-b91f370c41f0 // indirect
	github.com/mfridman/interpolate v0.0.2 // indirect
	github.com/pressly/goose/v3 v3.24.3 // indirect
	github.com/qdm12/reprint v0.0.0-20200326205758-722754a53494 // indirect
	github.com/sethvargo/go-retry v0.3.0 // indirect
	github.com/stephenafamo/scan v0.6.2 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sync v0.14.0 // indirect
)
