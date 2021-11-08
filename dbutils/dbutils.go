package dbutils

import (
	"database/sql"
	"strings"

	"github.com/juju/errors"
	. "github.com/volatiletech/sqlboiler/v4/queries/qm"
)

func ParseSorting(data []string, mapping map[string]string) ([]string, error) {
	sortList := []string{}
	for _, elem := range data {
		direction := " ASC"
		if strings.HasPrefix(elem, "-") {
			direction = " DESC"
			elem = elem[1:]
		}
		if _, ok := mapping[elem]; !ok {
			return nil, errors.Errorf("Attribute %s not found", elem)
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

func ParseFilters(attribute string, data string, mapping map[string]string) (QueryMod, error) {
	if _, ok := mapping[attribute]; !ok {
		return nil, errors.Errorf("Attribute %s not found", attribute)
	}
	d := strings.SplitN(data, ":", 2)
	if _, ok := filterMap[d[0]]; !ok {
		return nil, errors.Errorf("Operation %s not valid", d[0])
	}
	if d[0] == "isNull" || d[0] == "isNotNull" {
		return Where(mapping[attribute] + filterMap[d[0]]), nil
	}
	if len(d) < 2 {
		return nil, errors.Errorf("Invalid format data: %s", data)
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

// CountElem return the total number of elements
func CountElem(db *sql.DB, table string, where *string) (int, error) {
	if where == nil {
		tmp := ""
		where = &tmp
	}
	var number int
	err := db.QueryRow("SELECT COUNT(*) FROM " + table + " " + *where).Scan(&number)
	if err != nil {
		return -1, errors.Annotatef(err, table+" does not exists")
	}

	return number, nil
}

// ExistID check if the  id exists or not into the specified table
func ExistID(db *sql.DB, id string, table string) (bool, error) {
	row, err := db.Query("SELECT * FROM " + table + " WHERE [id] = " + id)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, errors.Annotatef(err, table+" does not exists")
	}

	return row.Next(), nil
}
