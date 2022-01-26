package dbutils

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"runtime/debug"
	"strings"

	"github.com/pressly/goose/v3"
	"github.com/volatiletech/sqlboiler/v4/boil"
	. "github.com/volatiletech/sqlboiler/v4/queries/qm"
	"gitlab.com/top-solution/go-libs/config"
)

// ErrEmptySort is raised when ParseSorting is called with an empty slice
// You should either handle it or use AddSorting instead
var ErrEmptySort = errors.New("at least a sort parameter is required")

// FilterMap maps the "public" name of an attribute with a DB column
type FilterMap map[string]string

// ParseSorting generates an OrderBy QueryMod starting from a given list of user-inputted values and an attribute->column map
// The user values should look like "field" (ASC) or "-field" (DESC)
func (f FilterMap) ParseSorting(sort []string) (QueryMod, error) {
	if len(sort) == 0 {
		return nil, nil
	}
	sortList := []string{}
	for _, elem := range sort {
		direction := " ASC"
		if strings.HasPrefix(elem, "-") {
			direction = " DESC"
			elem = elem[1:]
		}
		if _, ok := f[elem]; !ok {
			return nil, fmt.Errorf("Attribute %s not found", elem)
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

// WhereFilters map user-given operators to Where operators
var WhereFilters = map[string]string{
	"eq":        " = ?",
	"neq":       " != ?",
	"like":      " LIKE ? ESCAPE '_'",
	"notlike":   " NOT LIKE ? ESCAPE '_'",
	"lt":        " < ?",
	"lte":       " <= ?",
	"gt":        " > ?",
	"gte":       " >= ?",
	"isNull":    " IS NULL ",
	"isNotNull": " IS NOT NULL ",
	"in":        " IN ?",
	"notIn":     " NOT IN ?",
}

// ParseFilters generates an sqlboiler's QueryMod starting from an user-inputted attribute, user-inputted data, and an attribute->column map
// It also returns the parsed operator and value
func (f FilterMap) ParseFilters(attribute string, data string) (QueryMod, string, interface{}, error) {
	if _, ok := f[attribute]; !ok {
		return nil, "", nil, fmt.Errorf("Attribute %s not found", attribute)
	}
	d := strings.SplitN(data, ":", 2)
	if _, ok := WhereFilters[d[0]]; !ok {
		return nil, d[0], nil, fmt.Errorf("Operation %s not valid", d[0])
	}
	if d[0] == "isNull" || d[0] == "isNotNull" {
		return Where(f[attribute] + WhereFilters[d[0]]), d[0], nil, nil
	}
	if len(d) < 2 {
		return nil, d[0], nil, fmt.Errorf("Invalid format data: %s", data)
	}
	if d[0] == "in" || d[0] == "notIn" {
		var value []interface{}
		stringValue := strings.Split(d[1], ",")
		for _, v := range stringValue {
			value = append(value, v)
		}
		if d[0] == "in" {
			return WhereIn(f[attribute]+WhereFilters[d[0]], value...), d[0], value, nil
		}
		return WhereNotIn(f[attribute]+WhereFilters[d[0]], value...), d[0], value, nil
	}
	return Where(f[attribute]+WhereFilters[d[0]], d[1]), d[0], d[1], nil
}

// AddFilters adds the parsed filters to the query
func (f FilterMap) AddFilters(query *[]QueryMod, attribute string, data string) (err error) {
	mod, _, _, err := f.ParseFilters(attribute, data)
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
		return nil, errors.New("Invalid pagination parameters")
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

// Transaction wraps a function within an SQL transaction, that can be used to run multiple statements in a safe way
// In case of errors or panics, the transaction will be rolled back
func Transaction(db boil.Beginner, txFunc func(*sql.Tx) error) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer func() {
		//nolint:gocritic
		if p := recover(); p != nil {
			err = tx.Rollback()
			if err != nil {
				panic(p)
			}
			log.Printf("%s: %s", p, debug.Stack())
			err = errors.New("transaction failed")
		} else if err != nil {
			rollbackErr := tx.Rollback() // err is non-nil; don't change it
			log.Println(rollbackErr)
		} else {
			err = tx.Commit() // err is nil; if Commit returns an error, update err
		}
	}()
	err = txFunc(tx)
	return err
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
	case "postgres":
		connectionString = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			conf.Server, conf.Port, conf.User, conf.Password, conf.DB)
	}

	// Init Goose
	err := goose.SetDialect(conf.Driver)
	if err != nil {
		return nil, -1, fmt.Errorf("set migrations dialect: %w", err)
	}

	goose.SetBaseFS(fsys)

	currentVersion := int64(-1)

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

		currentVersion, err = goose.GetDBVersion(db)
		if err != nil {
			return nil, -1, fmt.Errorf("get db version: %w", err)
		}

		err = goose.Up(db, conf.Migrations.Path)
		if err != nil {
			return nil, -1, fmt.Errorf("migrate db: %w", err)
		}

		db.Close()
	}

	// sqlboiler wants to use the "mssql" driver, never "sqlserver"
	db, err := sql.Open(conf.Driver, connectionString)
	if err != nil {
		return nil, -1, fmt.Errorf("open db: %w", err)
	}

	// Make sure the DB is actually reachable
	err = db.Ping()
	if err != nil {
		return nil, -1, fmt.Errorf("pinging DB server: %w", err)
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
