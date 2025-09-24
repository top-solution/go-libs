package boilerops

import (
	"errors"
	"strings"

	. "github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/lib/pq"
	"github.com/top-solution/go-libs/v2/dbutils"
	"github.com/top-solution/go-libs/v2/dbutils/ops"
)

// NewBoilFilterMap creates a new FilterMap for sqlboiler's QueryMods
func NewBoilFilterMap(fields map[string]string) ops.FilterMap[QueryMod] {
	return ops.NewFilterMap(fields, &BoilFilterer{})
}

// BoilFilterer is a ops.Filterer for sqlboiler's QueryMods
type BoilFilterer struct{}

func (b *BoilFilterer) ParseFilter(filter, alias string, op string, rawValue string, having bool) (QueryMod, string, interface{}, error) {
	queryMod := Where
	if having {
		queryMod = Having
	}
	if ops.IsUnaryOp(op) {
		q := strings.ReplaceAll(filter, "{}", alias)
		return queryMod(q), q, nil, nil
	}
	if op == "in" || op == "notIn" {
		var value []interface{}
		stringValue := strings.Split(rawValue, ",")
		for _, v := range stringValue {
			value = append(value, strings.TrimSpace(v))
		}
		if dbutils.CurrentDriver == dbutils.PostgresDriver {
			q := strings.ReplaceAll(filter, "{}", alias)
			return queryMod(q, pq.Array(value)), q, pq.Array(value), nil
		}
		// FIXME: no support of non-postgres In/NotIn Having for MSSQL
		if op == "in" {
			q := strings.ReplaceAll(filter, "{}", alias)
			return WhereIn(q, value...), q, value, nil
		}
		q := strings.ReplaceAll(filter, "{}", alias)
		return WhereNotIn(q, value...), q, value, nil
	}
	q := strings.ReplaceAll(filter, "{}", alias)
	return queryMod(q, rawValue), q, rawValue, nil
}

func (b *BoilFilterer) ParseSorting(sortList []string) (QueryMod, error) {
	return OrderBy(strings.Join(sortList, ", ")), nil
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
