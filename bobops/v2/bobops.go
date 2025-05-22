package bobops

import (
	"strings"

	"github.com/lib/pq"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/top-solution/go-libs/dbutils/v2/ops"
)

// NewBoilFilterMap creates a new FilterMap for sqlboiler's QueryMods
func NewBobFilterMap(fields map[string]string) ops.FilterMap[bob.Mod[*dialect.SelectQuery]] {
	return ops.NewFilterMap(fields, &BobFilterer{})
}

type BobFilterer struct{}

func (b *BobFilterer) ParseFilter(filter, alias string, op string, rawValue string, having bool) (bob.Mod[*dialect.SelectQuery], string, interface{}, error) {
	if having {
		if ops.IsUnaryOp(op) {
			q := strings.ReplaceAll(filter, "{}", alias)
			return sm.Having(psql.Raw(q)), q, nil, nil
		}
		if op == "in" || op == "notIn" {
			var value []interface{}
			stringValue := strings.Split(rawValue, ",")
			for _, v := range stringValue {
				value = append(value, v)
			}

			q := strings.ReplaceAll(filter, "{}", alias)
			return sm.Having(psql.Raw(q, pq.Array(value))), q, pq.Array(value), nil
		}
		q := strings.ReplaceAll(filter, "{}", alias)
		return sm.Having(psql.Raw(q, rawValue)), q, rawValue, nil
	}
	if ops.IsUnaryOp(op) {
		q := strings.ReplaceAll(filter, "{}", alias)
		return sm.Where(psql.Raw(q)), q, nil, nil
	}
	if op == "in" || op == "notIn" {
		var value []interface{}
		stringValue := strings.Split(rawValue, ",")
		for _, v := range stringValue {
			value = append(value, v)
		}

		q := strings.ReplaceAll(filter, "{}", alias)
		return sm.Where(psql.Raw(q, pq.Array(value))), q, pq.Array(value), nil

	}

	q := strings.ReplaceAll(filter, "{}", alias)
	return sm.Where(psql.Raw(q, rawValue)), q, rawValue, nil
}

func (b *BobFilterer) ParseSorting(sortList []string) (bob.Mod[*dialect.SelectQuery], error) {
	return sm.OrderBy(strings.Join(sortList, ", ")), nil
}
