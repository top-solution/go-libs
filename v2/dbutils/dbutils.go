package dbutils

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"strings"
	"time"

	"github.com/pressly/goose/v3"
)

type DriverType string

const (
	MSSQLDriver    DriverType = "sqlserver"
	PostgresDriver DriverType = "postgres"
)

var CurrentDriver = PostgresDriver

// TxKey holds a transaction in a ctx
var TxKey txctx = "transaction"

var connectionRetries = []time.Duration{1, 1, 2, 2, 3, 5, 8}

type txctx string

// Beginner begins transactions.
type Beginner interface {
	Begin() (*sql.Tx, error)
}

// Executor can perform SQL queries.
type Executor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

// ContextExecutor can perform SQL queries with context
type ContextExecutor interface {
	Executor

	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// BeginnerExecutor is a combination of Beginner and ContextExecutor
type BeginnerExecutor interface {
	Beginner
	ContextExecutor
}

// DBConfig is a default config struct used to connect to a database
type DBConfig struct {
	// Driver is the driver name
	Driver string `yaml:"driver" conf:"help:The db driver name"`
	// Server is db host address
	Server string `yaml:"server" conf:"help:The db host"`
	// Port is the db port
	Port int `yaml:"port" conf:"help:The db name"`
	// User is the db user
	User string `yaml:"user" conf:"help:The db user"`
	// Password is the password for the db user
	Password string `yaml:"password" conf:"help:The db user password"`
	// DB is the db name
	DB         string `yaml:"db" conf:"help:The name of the DB"`
	Migrations struct {
		Run  bool   `yaml:"run" conf:"default:false,help:If true, migrations will be run on app startup"`
		Path string `yaml:"path" conf:"default:sql,help:The path to the directory containing the Goose-compatible SQL migrations"`
	} `yaml:"migrations"`
	// If connecting to an instance instead of a port
	Instance string `yaml:"instance" conf:"help:The db instance"`
	// Schema is the db schema
	Schema string `yaml:"schema" conf:"help:The db schema"`
}

// Transaction either embeds the transaction in the given context or uses an existing one from the context
func Transaction(ctx context.Context, db BeginnerExecutor, txFunc func(ctx context.Context, tx *sql.Tx) error) error {
	_, err := TransactionResult(ctx, db, func(ctx context.Context, tx *sql.Tx) (any, error) {
		return nil, txFunc(ctx, tx)
	})
	return err
}

// TransactionResult is the same as Transaction, but it returns a result along with the error
func TransactionResult[T any](ctx context.Context, db BeginnerExecutor, txFunc func(ctx context.Context, tx *sql.Tx) (T, error)) (result T, err error) {
	tx := Tx(ctx)
	if tx != nil {
		return txFunc(ctx, tx)
	}

	// No tx was found: start a new one and handle it
	tx, err = db.Begin()
	if err != nil {
		return
	}
	ctx = WithTx(ctx, tx)

	defer func() {
		//nolint:gocritic
		if p := recover(); p != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				panic(rollbackErr)
			}
			panic(p) // re-raise panic after Rollback
		} else if err != nil {
			rollbackErr := tx.Rollback() // err is non-nil; don't change it
			if rollbackErr != nil {
				err = fmt.Errorf("rollback failed (%s): %w", rollbackErr.Error(), err)
			}
		} else {
			err = tx.Commit() // err is nil; if Commit returns an error, update err
		}
	}()
	return txFunc(ctx, tx)
}

// WithTx enriches a context with a transaction
func WithTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, TxKey, tx)
}

// TxOr extracts a transaction from a context, with a fallback executor
func TxOr(ctx context.Context, fallback ContextExecutor) ContextExecutor {
	tx := Tx(ctx)
	if tx == nil {
		return fallback
	}
	return tx
}

// Tx extracts a transaction from a context, returns nil if no transaxction is found
func Tx(ctx context.Context) *sql.Tx {
	tx, ok := ctx.Value(TxKey).(*sql.Tx)
	if !ok {
		return nil
	}
	return tx
}

// DB is a wrapper for *sql.DB, providing a few utilities to handle migrations
type DB struct {
	*sql.DB
	conf DBConfig
	fsys fs.FS
}

// Open opens a database connection given a config struct
// It expects a fs.FS in order to fetch and run the DB migrations
// If you don't need them, just pass nil instead
func Open(conf DBConfig, fsys fs.FS) (db *DB, err error) {
	if conf.Driver == "" {
		return nil, errors.New("no SQL driver specified: please use one of [mssql,postgres]")
	}

	connectionString := fromDBConfToConnectionString(conf)

	if connectionString == "" {
		return nil, errors.New("unsupported driver: " + conf.Driver)
	}

	var sqlDB *sql.DB

	// Make sure the DB is actually reachable
	for _, delay := range connectionRetries {
		sqlDB, err = sql.Open(conf.Driver, connectionString)
		if err == nil {
			err = sqlDB.Ping()
			if err == nil {
				break
			}
		}
		time.Sleep(delay * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("reaching DB server: %w", err)
	}

	db = &DB{DB: sqlDB, conf: conf, fsys: fsys}
	// DB should be ready, run migrations if needed
	if conf.Migrations.Run {
		if fsys != nil {
			err = db.Migrate(fsys)
			if err != nil {
				return nil, fmt.Errorf("cannote run db migrations: %w", err)
			}
		}
	}

	return db, nil
}

func (d *DB) Migrate(fsys fs.FS) (err error) {
	if !d.conf.Migrations.Run {
		return nil
	}
	// Init Goose
	err = goose.SetDialect(d.conf.Driver)
	if err != nil {
		return fmt.Errorf("set migrations dialect: %w", err)
	}
	d.fsys = fsys
	goose.SetBaseFS(d.fsys)

	tableName := goose.DefaultTablename

	// Specify schema when hinted by the configuration
	if d.conf.Schema != "" {
		schema, _, _ := strings.Cut(d.conf.Schema, ",")
		tableName = schema + "." + tableName
	}

	goose.SetTableName(tableName)

	// Goose wants to use the "sqlserver" driver, never "mssql"
	driver := d.conf.Driver
	if driver == "mssql" {
		driver = "sqlserver"
	}

	err = goose.Up(d.DB, d.conf.Migrations.Path)
	if err != nil {
		return fmt.Errorf("migrate db: %w", err)
	}

	return nil
}

// Convert the database configuration to connection string
func fromDBConfToConnectionString(conf DBConfig) string {
	query := url.Values{}

	u := &url.URL{
		Scheme: conf.Driver,
		Host:   conf.Server,
	}
	if conf.Port != 0 {
		u.Host += fmt.Sprintf(":%d", conf.Port)
	}

	switch conf.Driver {
	case string(MSSQLDriver):
		query.Add("database", conf.DB)
		CurrentDriver = MSSQLDriver
	case string(PostgresDriver):
		query.Add("dbname", conf.DB)
		query.Add("sslmode", "disable")
		if conf.Schema != "" {
			query.Add("options", "-c search_path="+conf.Schema)
		}
		CurrentDriver = PostgresDriver
	default:
		return ""
	}

	if conf.User != "" {
		u.User = url.UserPassword(conf.User, conf.Password)
	}

	if conf.Instance != "" {
		u.Path = conf.Instance
	}

	u.RawQuery = query.Encode()
	connectionString := u.String()

	return connectionString
}

// Up runs the migrations up to the latest version
func (d *DB) Up() error {
	if d.fsys == nil {
		return errors.New("can't run migrations: no file system was passed to Open()")
	}

	err := goose.Up(d.DB, d.conf.Migrations.Path)
	if err != nil {
		return fmt.Errorf("running db migrations: %w", err)
	}
	return nil
}

// Version return the current DB version
func (d *DB) Version() (int64, error) {
	if d.fsys == nil {
		return -1, nil
	}
	return goose.GetDBVersion(d.DB)
}
