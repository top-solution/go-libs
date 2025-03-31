package ops

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/top-solution/go-libs/v2/dbutils"
)

// ErrEmptySort is raised when ParseSorting is called with an empty slice
// You should either handle it or use AddSorting instead
var ErrEmptySort = errors.New("at least a sort parameter is required")

// UnaryOps is a list of operators which don't require values
var UnaryOps = []string{"isNull", "isNotNull", "isEmpty", "isNotEmpty"}

// IsUnaryOp returns true if the operator is unary (no value required)
func IsUnaryOp(op string) bool {
	return slices.Contains(UnaryOps, op)
}

type Filterer[T any] interface {
	// Sort returns the sorting string for the given attribute
	ParseFilter(filter, alias string, op string, rawValue string, having bool) (T, string, interface{}, error)
	ParseSorting(sortList []string) (T, error)
}

// FilterMap is a helper struct to parse filters into a slice of query mods
// Query Mods can be from different query builders
type FilterMap[T any] struct {
	filterer Filterer[T]
	fields   map[string]string
}

// NewFilterMap creates a new FilterMap
// If you need to use this with sqlboiler, see boilerops package
// If you need to use this with bob, see bobops package
func NewFilterMap[T any](fields map[string]string, f Filterer[T]) FilterMap[T] {
	return FilterMap[T]{
		filterer: f,
		fields:   fields,
	}
}

// AddFilters parses the filters and adds them to the given list of query mods
func (f FilterMap[T]) AddFilters(q *[]T, attribute string, filters ...string) error {
	filter, _, _, _, err := parseFilters(f.filterer, f.fields, attribute, false, filters...)
	if err != nil {
		return fmt.Errorf("error parsing filters: %w", err)
	}
	*q = append(*q, filter...)
	return nil
}

// AddHavingFilters parses the filters and adds them to the given list of query mods as Having clauses
func (f FilterMap[T]) AddHavingFilters(query *[]T, attribute string, data ...string) (err error) {
	qmods, _, _, _, err := f.ParseFilters(attribute, true, data...)
	if err != nil {
		return err
	}
	*query = append(*query, qmods...)
	return nil
}

// ParseFilters parses the filters and returns the query mods, raw queries, operators and values
func (f FilterMap[T]) ParseFilters(attribute string, having bool, filters ...string) ([]T, []string, []string, []interface{}, error) {
	return parseFilters(f.filterer, f.fields, attribute, having, filters...)
}

// ParseSorting generates an OrderBy QueryMod starting from a given list of user-inputted values and an attribute->column map
// The user values should look like "field" (ASC) or "-field" (DESC)
func (f FilterMap[T]) ParseSorting(sort []string) (T, error) {
	if len(sort) == 0 {
		return *new(T), ErrEmptySort
	}
	sortList := []string{}
	for _, elem := range sort {
		direction := " ASC"
		if strings.HasPrefix(elem, "-") {
			direction = " DESC"
			elem = elem[1:]
		}
		if _, ok := f.fields[elem]; !ok {
			return *new(T), fmt.Errorf("attribute %s not found", elem)
		}
		sortList = append(sortList, f.fields[elem]+direction)
	}
	return f.filterer.ParseSorting(sortList)
}

// AddSorting adds the result of ParseSorting to a given query
func (f FilterMap[T]) AddSorting(query *[]T, sort []string) (err error) {
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

func parseFilters[T any](filterer Filterer[T], f map[string]string, attribute string, having bool, filters ...string) ([]T, []string, []string, []interface{}, error) {
	var qmods []T
	var rawQueries []string
	var ops []string
	var vals []interface{}

	if _, ok := f[attribute]; !ok {
		return nil, nil, nil, nil, fmt.Errorf("attribute %s not found", attribute)
	}

	driverFilters := postgresWhereFilters
	if dbutils.CurrentDriver == dbutils.MSSQLDriver {
		driverFilters = msSQLWhereFilters
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
		if _, ok := driverFilters[op]; !ok {
			return nil, nil, nil, nil, fmt.Errorf("operation %s is not implemented", op)
		}
		qmod, raw, val, err := filterer.ParseFilter(driverFilters[op], f[attribute], op, rawValue, having)
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

var msSQLWhereFilters = map[string]string{
	"eq":         "{} = ?",
	"neq":        "{} != ?",
	"like":       "{} LIKE ? ESCAPE '_'",
	"notLike":    "{} NOT ILIKE ? ESCAPE '_' OR {} IS NULL",
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

var postgresWhereFilters = map[string]string{
	"eq":         "{} = ?",
	"neq":        "{} != ?",
	"like":       "{} ILIKE ? ESCAPE '_'",
	"notLike":    "{} NOT ILIKE ? ESCAPE '_' OR {} IS NULL",
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
