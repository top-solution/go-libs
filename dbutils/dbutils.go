package dbutils

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"runtime/debug"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/prometheus/common/log"
	"github.com/top-solution/go-libs/config"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	. "github.com/volatiletech/sqlboiler/v4/queries/qm"
	"golang.org/x/exp/slices"
)

// TxKey holds a transaction in a ctx
var TxKey txctx = "transaction"

type txctx string

// ErrEmptySort is raised when ParseSorting is called with an empty slice
// You should either handle it or use AddSorting instead
var ErrEmptySort = errors.New("at least a sort parameter is required")

var connectionRetries = []time.Duration{1, 1, 2, 2, 3, 5, 8}

// UnaryOps is a list of operators which don't require values
var UnaryOps = []string{"isNull", "isNotNull", "isEmpty", "isNotEmpty"}

// QueryMods is an helper that allows treating arrays of QueryMod as a single QueryMod
type QueryMods []QueryMod

func (m QueryMods) Apply(q *queries.Query) {
	Apply(q, m...)
}

// FilterMap maps the "public" name of an attribute with a DB column
type FilterMap map[string]string

// ParseSorting generates an OrderBy QueryMod starting from a given list of user-inputted values and an attribute->column map
// The user values should look like "field" (ASC) or "-field" (DESC)
func (f FilterMap) ParseSorting(sort []string) (QueryMod, error) {
	if len(sort) == 0 {
		return nil, ErrEmptySort
	}
	sortList := []string{}
	for _, elem := range sort {
		direction := " ASC"
		if strings.HasPrefix(elem, "-") {
			direction = " DESC"
			elem = elem[1:]
		}
		if _, ok := f[elem]; !ok {
			return nil, fmt.Errorf("attribute %s not found", elem)
		}
		sortList = append(sortList, f[elem]+direction)
	}
	return OrderBy(strings.Join(sortList, ", ")), nil
}

// AddSorting adds the result of ParseSorting to a given query
func (f FilterMap) AddSorting(query *[]QueryMod, sort []string) (err error) {
	mod, err := f.ParseSorting(sort)
	if err != nil {
		// If no sort parameters are passed, simply return the query as-is
		if errors.Is(err, ErrEmptySort) {
			return nil
		}
		return err
	}
	*query = append(*query, mod)
	return nil
}

type DriverType string

const (
	MSSQLDriver    DriverType = "mssql"
	PostgresDriver DriverType = "postgres"
)

var CurrentDriver = PostgresDriver

// WhereFilters map user-given operators to Where operators
var MSSQLWhereFilters = map[string]string{
	"eq":         "{} = ?",
	"neq":        "{} != ?",
	"like":       "{} LIKE ? ESCAPE '_'",
	"notLike":    "{} NOT LIKE ? ESCAPE '_' OR {} IS NULL",
	"lt":         "{} < ?",
	"lte":        "{} <= ?",
	"gt":         "{} > ?",
	"gte":        "{} >= ?",
	"isNull":     "{} IS NULL",
	"isNotNull":  "{} IS NOT NULL",
	"in":         "{} IN ?",
	"notIn":      "{} NOT IN ?",
	"isEmpty":    "coalesce({},'') = ''",
	"isNotEmpty": "coalesce({},'') != ''",
}

// WhereFilters map user-given operators to Where operators
var PostgresWhereFilters = map[string]string{
	"eq":         "{} = ?",
	"neq":        "{} != ?",
	"like":       "{} ILIKE ? ESCAPE '_'",
	"notLike":    "{} NOT LIKE ? ESCAPE '_' OR {} IS NULL",
	"lt":         "{} < ?",
	"lte":        "{} <= ?",
	"gt":         "{} > ?",
	"gte":        "{} >= ?",
	"isNull":     "{} IS NULL",
	"isNotNull":  "{} IS NOT NULL",
	"in":         "{} = ANY(?)",
	"notIn":      "{} != ALL(?)",
	"isEmpty":    "coalesce({},'') = ''",
	"isNotEmpty": "coalesce({},'') != ''",
}

var WhereFilters = PostgresWhereFilters

// ParseFilters generates an sqlboiler's QueryMod starting from an user-inputted attribute, user-inputted data, and an attribute->column map
// It also returns the parsed operator and value
func (f FilterMap) ParseFilters(attribute string, having bool, filters ...string) (QueryMod, []string, []string, []interface{}, error) {
	var qmods QueryMods
	var rawQueries []string
	var ops []string
	var vals []interface{}

	if _, ok := f[attribute]; !ok {
		return nil, nil, nil, nil, fmt.Errorf("attribute %s not found", attribute)
	}

	for _, filter := range filters {
		spl := strings.SplitN(filter, ":", 2)
		op := spl[0]
		rawValue := ""
		if len(spl) < 2 {
			if !IsUnaryOp(op) {
				return nil, nil, nil, nil, fmt.Errorf("operation %s is not valid", op)
			}
		} else {
			rawValue = spl[1]
		}
		if _, ok := WhereFilters[op]; !ok {
			return nil, nil, nil, nil, fmt.Errorf("operation %s is not implemented", op)
		}
		qmod, raw, val, err := f.parseFilter(attribute, op, rawValue, having)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		ops = append(ops, op)
		rawQueries = append(rawQueries, raw)
		qmods = append(qmods, qmod)
		vals = append(vals, val)
	}
	return qmods, rawQueries, ops, vals, nil
}

func (f FilterMap) parseFilter(attribute string, op string, rawValue string, having bool) (QueryMod, string, interface{}, error) {
	queryMod := qm.Where
	if having {
		queryMod = qm.Having
	}
	if IsUnaryOp(op) {
		q := strings.ReplaceAll(WhereFilters[op], "{}", f[attribute])
		return queryMod(q), q, nil, nil
	}
	if op == "in" || op == "notIn" {
		var value []interface{}
		stringValue := strings.Split(rawValue, ",")
		for _, v := range stringValue {
			value = append(value, v)
		}
		if CurrentDriver == PostgresDriver {
			q := strings.ReplaceAll(WhereFilters[op], "{}", f[attribute])
			return queryMod(q, pq.Array(value)), q, pq.Array(value), nil
		}
		// FIXME: no support of non-postgres In/NotIn Having for MSSQL
		if op == "in" {
			q := strings.ReplaceAll(WhereFilters[op], "{}", f[attribute])
			return WhereIn(q, value...), q, value, nil
		}
		q := strings.ReplaceAll(WhereFilters[op], "{}", f[attribute])
		return WhereNotIn(q, value...), q, value, nil
	}
	q := strings.ReplaceAll(WhereFilters[op], "{}", f[attribute])
	return queryMod(q, rawValue), q, rawValue, nil
}

// AddFilters adds the parsed filters to the query with a Where querymod
func (f FilterMap) AddFilters(query *[]QueryMod, attribute string, data ...string) (err error) {
	mod, _, _, _, err := f.ParseFilters(attribute, false, data...)
	if err != nil {
		return err
	}
	*query = append(*query, mod)
	return nil
}

// AddHavingFilters adds the parsed filters to the query with a Having QueryMod
func (f FilterMap) AddHavingFilters(query *[]QueryMod, attribute string, data ...string) (err error) {
	mod, _, _, _, err := f.ParseFilters(attribute, true, data...)
	if err != nil {
		return err
	}
	*query = append(*query, mod)
	return nil
}

// ParsePagination generates a Limit+Offset QueryMod slice given an user-inputted offset and limit
func ParsePagination(offset *int, limit *int) (res []QueryMod, err error) {
	res = []QueryMod{}
	if (limit != nil && offset == nil) || (limit == nil && offset != nil) {
		return nil, errors.New("invalid pagination parameters")
	}
	if limit != nil && offset != nil {
		res = append(res, Limit(*limit), Offset(*offset))
	}
	return res, nil
}

// AddPagination adds the parsed pagination filters to the query
func AddPagination(query *[]QueryMod, offset *int, limit *int) (err error) {
	mods, err := ParsePagination(offset, limit)
	if err != nil {
		return err
	}
	*query = append(*query, mods...)
	return nil
}

type BeginnerExecutor interface {
	boil.Beginner
	boil.Executor
}

// Transaction wraps a function within an SQL transaction, that can be used to run multiple statements in a safe way
// In case of errors or panics, the transaction will be rolled back
func Transaction(db BeginnerExecutor, txFunc func(tx *sql.Tx) error) (err error) {
	return TransactionCtx(context.TODO(), db, func(ctx context.Context) error {
		return txFunc(Tx(ctx))
	})
}

// TransactionCtx is the same as Transaction, but either embeds the transaction in the given context
// or uses an existing one from the context
func TransactionCtx(ctx context.Context, db BeginnerExecutor, txFunc func(ctx context.Context) error) (err error) {
	tx := Tx(ctx)
	if tx != nil {
		return txFunc(ctx)
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
				panic(p)
			}
			switch x := p.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = fmt.Errorf("transaction failed: %w", x)
			default:
				err = fmt.Errorf("transaction failed for unknown panic: %v", x)
			}
			log.Error("transaction failed", "err", err, "stack", string(debug.Stack()))
			err = fmt.Errorf("%v", err)
		} else if err != nil {
			rollbackErr := tx.Rollback() // err is non-nil; don't change it
			if rollbackErr != nil {
				err = fmt.Errorf("rollback failed (%s): %w", rollbackErr.Error(), err)
			}
		} else {
			err = tx.Commit() // err is nil; if Commit returns an error, update err
		}
	}()
	err = txFunc(ctx)
	return err
}

// WithTx enriches a context with a transaction
func WithTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, TxKey, tx)
}

// TxOr extracts a transaction from a context, with a fallback executor
func TxOr(ctx context.Context, fallback boil.Executor) boil.Executor {
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
	conf config.DBConfig
	fsys fs.FS
}

// Open opens a database connection given a config struct
// It expects a fs.FS in order to fetch and run the DB migrations
// If you don't need them, just pass nil instead
func Open(conf config.DBConfig, fsys fs.FS) (*DB, int64, error) {
	connectionString := ""

	if conf.Driver == "" {
		return nil, 0, errors.New("no SQL driver specified: please use one of [mssql,sqlserver,postgres]")
	}

	switch conf.Driver {
	case "mssql":
		if conf.User == "" {
			connectionString = fmt.Sprintf("server=%s;port=%d;database=%s",
				conf.Server, conf.Port, conf.DB)
		} else {
			connectionString = fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s",
				conf.User, conf.Password, conf.Server, conf.Port, conf.DB)
		}
		// FIXME: this means we can't have both a mssql and a postgres connections active at the same time
		WhereFilters = MSSQLWhereFilters
		CurrentDriver = MSSQLDriver
	case "postgres":
		connectionString = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			conf.Server, conf.Port, conf.User, conf.Password, conf.DB)
		// FIXME: this means we can't have both a mssql and a postgres connections active at the same time
		WhereFilters = PostgresWhereFilters
		CurrentDriver = PostgresDriver
	}

	// Init Goose
	err := goose.SetDialect(conf.Driver)
	if err != nil {
		return nil, -1, fmt.Errorf("set migrations dialect: %w", err)
	}

	goose.SetBaseFS(fsys)

	currentVersion := int64(-1)

	var db *sql.DB

	// Make sure the DB is actually reachable
	for _, delay := range connectionRetries {
		db, err = sql.Open(conf.Driver, connectionString)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		time.Sleep(delay * time.Second)
	}
	if err != nil {
		return nil, -1, fmt.Errorf("reaching DB server: %w", err)
	}

	// DB should be ready, run migrations if needed
	if conf.Migrations.Run {
		// Goose wants to use the "sqlserver" driver, never "mssql"
		driver := conf.Driver
		if driver == "mssql" {
			driver = "sqlserver"
		}

		db, err := sql.Open(driver, connectionString)
		if err != nil {
			return nil, -1, fmt.Errorf("open db for migrations: %w", err)
		}
		defer db.Close()

		currentVersion, err = goose.GetDBVersion(db)
		if err != nil {
			return nil, -1, fmt.Errorf("get db version: %w", err)
		}

		err = goose.Up(db, conf.Migrations.Path)
		if err != nil {
			return nil, -1, fmt.Errorf("migrate db: %w", err)
		}
	}

	return &DB{DB: db, conf: conf, fsys: fsys}, currentVersion, nil
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
		return -1, errors.New("can't get current version: no file system was passed to Open()")
	}
	return goose.GetDBVersion(d.DB)
}

// IsUnaryOp returns true if the operator is unary (no value required)
func IsUnaryOp(op string) bool {
	return slices.Contains(UnaryOps, op)
}
