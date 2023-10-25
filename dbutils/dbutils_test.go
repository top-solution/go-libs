package dbutils

import (
	"strings"
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/volatiletech/sqlboiler/v4/drivers"
	"github.com/volatiletech/sqlboiler/v4/queries"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

func NewQuery(mods ...qm.QueryMod) *queries.Query {
	q := &queries.Query{}
	// TODO: test MSSQL
	queries.SetDialect(q, &drivers.Dialect{
		LQ: 0x5b,
		RQ: 0x5d,
	})
	qm.Apply(q, mods...)

	return q
}

type filter struct {
	attr string
	raw  []string
}

func TestAddFilters(t *testing.T) {

	var filterMap = FilterMap{
		"id":               "id",
		"createDatetime":   "create_datetime",
		"customerSupplier": "customer_supplier",
	}

	tcs := []struct {
		name         string
		mods         []qm.QueryMod
		filters      []filter
		expectedSQL  string
		expectedVals interface{}
		expectedErr  bool
	}{
		{
			name: "in",
			mods: []qm.QueryMod{qm.Select("id", "name"), qm.From("tables")},
			filters: []filter{
				{
					attr: "id",
					raw:  []string{"in:ciao,come,va"},
				},
			},
			expectedSQL:  "SELECT [id], [name] FROM [tables] WHERE (id = ANY(?));",
			expectedVals: []any{pq.GenericArray{A: []any{"ciao", "come", "va"}}},
		},
		{
			name: "notIn",
			mods: []qm.QueryMod{qm.Select("id", "name"), qm.From("tables")},
			filters: []filter{
				{
					attr: "id",
					raw:  []string{"notIn:ciao,come,va"},
				},
			},
			expectedSQL:  "SELECT [id], [name] FROM [tables] WHERE (id != ALL(?));",
			expectedVals: []any{pq.GenericArray{A: []any{"ciao", "come", "va"}}},
		},
		{
			name: "isNull",
			mods: []qm.QueryMod{qm.Select("id", "name"), qm.From("tables")},
			filters: []filter{
				{
					attr: "id",
					raw:  []string{"isNull"},
				},
			},
			expectedSQL:  "SELECT [id], [name] FROM [tables] WHERE (id IS NULL);",
			expectedVals: []interface{}(nil),
		},
		{
			name: "isNotNull",
			mods: []qm.QueryMod{qm.Select("id", "name"), qm.From("tables")},
			filters: []filter{
				{
					attr: "id",
					raw:  []string{"isNotNull"},
				},
			},
			expectedSQL:  "SELECT [id], [name] FROM [tables] WHERE (id IS NOT NULL);",
			expectedVals: []interface{}(nil),
		},
		{
			name: "isEmpty",
			mods: []qm.QueryMod{qm.Select("id", "name"), qm.From("tables")},
			filters: []filter{
				{
					attr: "id",
					raw:  []string{"isEmpty"},
				},
			},
			expectedSQL:  "SELECT [id], [name] FROM [tables] WHERE (coalesce(id,'') = '');",
			expectedVals: []interface{}(nil),
		},
		{
			name: "isNotEmpty",
			mods: []qm.QueryMod{qm.Select("id", "name"), qm.From("tables")},
			filters: []filter{
				{
					attr: "id",
					raw:  []string{"isNotEmpty"},
				},
			},
			expectedSQL:  "SELECT [id], [name] FROM [tables] WHERE (coalesce(id,'') != '');",
			expectedVals: []interface{}(nil),
		},
		{
			name: "multipleFilterSameAttr",
			mods: []qm.QueryMod{qm.Select("id", "name"), qm.From("tables")},
			filters: []filter{
				{
					attr: "id",
					raw:  []string{"gt:a", "lt:b"},
				},
			},
			expectedSQL:  "SELECT [id], [name] FROM [tables] WHERE (id > ?) AND (id < ?);",
			expectedVals: []interface{}{"a", "b"},
		},
		{
			name: "multipleFilters",
			mods: []qm.QueryMod{qm.Select("id", "name"), qm.From("tables")},
			filters: []filter{
				{
					attr: "id",
					raw:  []string{"gt:a", "lt:b"},
				},
				{
					attr: "createDatetime",
					raw:  []string{"gt:2022-04-01", "lt:2023-04-01"},
				},
			},
			expectedSQL:  "SELECT [id], [name] FROM [tables] WHERE (id > ?) AND (id < ?) AND (create_datetime > ?) AND (create_datetime < ?);",
			expectedVals: []interface{}{"a", "b", "2022-04-01", "2023-04-01"},
		},
	}

	// test simple ops
	for f, v := range WhereFilters {
		if f == "isNull" || f == "isNotNull" || f == "in" || f == "notIn" || f == "isEmpty" || f == "isNotEmpty" {
			continue
		}

		tcs = append(tcs, struct {
			name         string
			mods         []qm.QueryMod
			filters      []filter
			expectedSQL  string
			expectedVals interface{}
			expectedErr  bool
		}{
			name: f,
			mods: []qm.QueryMod{qm.Select("id", "name"), qm.From("tables")},
			filters: []filter{
				{
					attr: "id",
					raw:  []string{f + ":ciao"},
				},
			},
			expectedSQL:  "SELECT [id], [name] FROM [tables] WHERE (" + strings.ReplaceAll(v, "{}", "id") + ");",
			expectedVals: []interface{}{"ciao"},
		})
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			for _, f := range tc.filters {
				err = filterMap.AddFilters(&tc.mods, f.attr, f.raw...)
				if err != nil {
					break
				}
			}

			if tc.expectedErr {
				assert.NotNil(t, err)
				return
			} else {
				assert.Equal(t, nil, err)
			}

			q := NewQuery(tc.mods...)
			sql, vals := queries.BuildQuery(q)

			assert.Equal(t, tc.expectedSQL, sql)
			assert.EqualValues(t, tc.expectedVals, vals)
		})
	}
}

// TODO: test more operators, test QueryMods
func TestParseFilters(t *testing.T) {
	var filterMap = FilterMap{
		"id":               "id",
		"createDatetime":   "create_datetime",
		"customerSupplier": "customer_supplier",
	}

	tcs := []struct {
		name         string
		having       bool
		filters      []filter
		expectedRaws []string
		expectedOps  []string
		expectedVals interface{}
		expectedErr  bool
	}{
		{
			name:   "in",
			having: false,
			filters: []filter{
				{
					attr: "id",
					raw:  []string{"in:ciao,come,va"},
				},
			},
			expectedRaws: []string{"id = ANY(?)"},
			expectedOps:  []string{"in"},
			expectedVals: []any{[]any{"ciao", "come", "va"}},
		},
		{
			name:   "notIn",
			having: false,
			filters: []filter{
				{
					attr: "id",
					raw:  []string{"notIn:ciao,come,va"},
				},
			},
			expectedRaws: []string{"id != ALL(?)"},
			expectedOps:  []string{"notIn"},
			expectedVals: []any{[]any{"ciao", "come", "va"}},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			for _, f := range tc.filters {
				_, raws, ops, vals, err := filterMap.ParseFilters(f.attr, tc.having, f.raw...)
				if err != nil {
					break
				}
				if tc.expectedErr {
					assert.NotNil(t, err)
					return
				} else {
					assert.Equal(t, nil, err)
				}

				assert.EqualValues(t, tc.expectedOps, ops)
				assert.EqualValues(t, tc.expectedRaws, raws)
				assert.EqualValues(t, tc.expectedVals, vals)
			}
		})
	}
}
