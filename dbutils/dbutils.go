package dbutils

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"strings"

	"github.com/volatiletech/sqlboiler/v4/boil"
	. "github.com/volatiletech/sqlboiler/v4/queries/qm"
)

// ParseSorting generates an OrderBy QueryMod slice starting from a given list of user-inputted values and an attribute->column map
// The user values should look like "field" (ASC) or "-field" (DESC)
//
// An empty slice is returned when no filters are applied, so it's convenient to always append to the current query
func ParseSorting(data []string, mapping map[string]string) ([]QueryMod, error) {
	sortList := []string{}
	for _, elem := range data {
		direction := " ASC"
		if strings.HasPrefix(elem, "-") {
			direction = " DESC"
			elem = elem[1:]
		}
		if _, ok := mapping[elem]; !ok {
			return nil, fmt.Errorf("Attribute %s not found", elem)
		}
		sortList = append(sortList, mapping[elem]+direction)
	}
	if len(sortList) == 0 {
		return nil, nil
	}
	return []QueryMod{OrderBy(strings.Join(sortList, ", "))}, nil
}

var filterMap = map[string]string{
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
func ParseFilters(attribute string, data string, mapping map[string]string) (QueryMod, error) {
	if _, ok := mapping[attribute]; !ok {
		return nil, fmt.Errorf("Attribute %s not found", attribute)
	}
	d := strings.SplitN(data, ":", 2)
	if _, ok := filterMap[d[0]]; !ok {
		return nil, fmt.Errorf("Operation %s not valid", d[0])
	}
	if d[0] == "isNull" || d[0] == "isNotNull" {
		return Where(mapping[attribute] + filterMap[d[0]]), nil
	}
	if len(d) < 2 {
		return nil, fmt.Errorf("Invalid format data: %s", data)
	}
	if d[0] == "in" || d[0] == "notIn" {
		var value []interface{}
		stringValue := strings.Split(d[1], ",")
		for _, v := range stringValue {
			value = append(value, v)
		}
		if d[0] == "in" {
			return WhereIn(mapping[attribute]+filterMap[d[0]], value...), nil
		}
		return WhereNotIn(mapping[attribute]+filterMap[d[0]], value...), nil
	}
	return Where(mapping[attribute]+filterMap[d[0]], d[1]), nil
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

// AddPagination is DEPRECATED for name consistency: use ParsePagination instead
func AddPagination(offset *int, limit *int) (res []QueryMod, err error) {
	return ParsePagination(offset, limit)
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

// DBConfig is a default config struct used to connect to a database
type DBConfig struct {
	// Driver contains the driver name
	Driver string `yaml:"driver"`
	// Type contains the DB type: it's a MSSQL thing
	Type string `yaml:"type"`
	// Server contains the db host address
	Server string `yaml:"server"`
	// Port contains the db port
	Port int `yaml:"port"`
	// User contaisn the user to access the db
	User string `yaml:"user"`
	// Password contains the password to access the db
	Password string `yaml:"password"`
	// DB contains the DB name
	DB string `yaml:"db"`
	// MigrationsPath contains the path for the migration sql files
	MigrationsPath string `yaml:"migrations_path"`
}

// DB is a wrapper for sql.DB, reserved for future migration utilities
// It can be used as a regular *sql.DB
type DB struct {
	*sql.DB
}

// Open opens a database connection given a config struct
func Open(conf *DBConfig) (*DB, error) {
	connectionString := ""

	switch conf.Driver {
	case "mssql", "":
		if conf.User == "" {
			connectionString = fmt.Sprintf("server=%s;port=%d;database=%s",
				conf.Server, conf.Port, conf.DB)
		} else {
			connectionString = fmt.Sprintf("%s://%s:%s@%s:%d?database=%s",
				conf.Type, conf.User, conf.Password, conf.Server, conf.Port, conf.DB)
		}
	case "postgres":
		connectionString = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			conf.Server, conf.Port, conf.User, conf.Password, conf.DB)
	}

	db, err := sql.Open(conf.Driver, connectionString)
	if err != nil {
		return nil, (err)
	}

	return &DB{DB: db}, nil
}
