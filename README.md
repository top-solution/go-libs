# go-libs

A collection of Go utility packages for common development tasks.

## Packages

- **authorizer** - Policy-based authorization using Ladon with custom conditions
- **config** - Configuration file parsing with flags and environment variable support
- **dbutils** - Database utilities including migrations, transactions, and Bob/SQLBoiler ORM helpers (see also [generator docs](dbutils/ops/gen/README.md))
- **email** - SMTP email sending with HTML template support
- **fs** - Filesystem utilities including fallback filesystem for SPAs
- **humautils** - Utilities for Huma API framework including response helpers and endpoint registration
- **keys** - JWT token handling and middleware
- **logging** - Structured logging configuration with multiple output formats
- **middlewares** - HTTP middleware collection (CORS, caching, etc.)
- **policies** - Role-based access control integration
- **version** - Build version information helpers
