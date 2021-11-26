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

// ParseSorting generates a raw SORT clause starting from a given list of user-inputted values and an attribute->column map
// The user values should look like "field" (ASC) or "-field" (DESC)
func ParseSorting(data []string, mapping map[string]string) ([]string, error) {
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
	return sortList, nil
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
