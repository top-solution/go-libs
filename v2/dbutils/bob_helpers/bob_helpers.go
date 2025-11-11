package bob_helpers

import (
	"database/sql"
	"reflect"
	"time"

	"github.com/aarondl/opt"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/scan"
)

// Ptr returns a pointer to the given value.
func Ptr[T any](v T) *T {
	return &v
}

// NowPtr returns a pointer to the current time.Time.
func NowPtr() *time.Time {
	n := time.Now()
	return &n
}

// Null returns a sql.Null[T] that is Valid=false if v is zero value.
func Null[T any](v T) sql.Null[T] {
	return sql.Null[T]{
		V:     v,
		Valid: !isZero(v),
	}
}

// NullPtr returns a pointer to a sql.Null[T].
// It sets Valid=false if v is the zero value of T.
func NullPtr[T any](v T) *sql.Null[T] {
	return &sql.Null[T]{
		V:     v,
		Valid: !isZero(v),
	}
}

// NullPtrFromPtr returns a pointer to a sql.Null[T] based on the provided *T.
// If the input pointer is nil, it returns &sql.Null[T]{} “unset” Null[T].
// Otherwise, it returns &sql.Null[T]{V: *v, Valid: true}.
func NullPtrFromPtr[T any](v *T) *sql.Null[T] {
	if v == nil {
		return &sql.Null[T]{}
	}
	return &sql.Null[T]{
		V:     *v,
		Valid: true,
	}
}

// isZero checks whether v is the zero value of its type.
func isZero[T any](v T) bool {
	// Special case for bool: always consider it non-zero (i.e., always valid)
	switch any(v).(type) {
	case bool:
		return false
	}
	var zero T
	return reflect.DeepEqual(v, zero)
}

// IncludeSubqueryAsCTE appends a subquery as a Common Table Expression (CTE)
// to the provided query modifiers slice.
//
// This helper simplifies adding a named subquery (`WITH <alias> AS (...)`) to a
// `bob`-built query, allowing you to keep query composition modular.
//
// Example:
//
//	var mods []bob.Mod[*dialect.SelectQuery]
//	sub := []bob.Mod[*dialect.SelectQuery]{
//		psql.From("users").Where(psql.Col("active").Eq(true)),
//	}
//	IncludeSubqueryAsCTE(&mods, sub, "active_users")
//
// Generates:
//
//	WITH active_users AS (
//	    SELECT * FROM users WHERE active = true
//	)
//
// Parameters:
//   - q: pointer to the main query modifiers slice to be extended.
//   - subQuery: slice of modifiers defining the subquery to include.
//   - alias: name of the CTE (the identifier after WITH).
func IncludeSubqueryAsCTE(q *[]bob.Mod[*dialect.SelectQuery], subQuery []bob.Mod[*dialect.SelectQuery], alias string) {
	sub := psql.Select(
		subQuery...,
	)
	*q = append(*q,
		sm.With(alias).As(sub),
	)
}

// TableWithPrefix prefixes all columns in the given ColumnsExpr with the table alias.
// Equivalent to using `--prefix:<alias>.` in bob SQL files.
func TableWithPrefix(alias string, col expr.ColumnsExpr) expr.ColumnsExpr {
	return col.WithPrefix(alias + ".")
}

// TableWithPrefixAndParent prefixes all columns and sets the parent to the alias.
// Useful for nested structs when mapping joined tables.
func TableWithPrefixAndParent(alias string, col expr.ColumnsExpr) expr.ColumnsExpr {
	return col.WithPrefix(alias + ".").WithParent(alias)
}

// GroupByWithParent groups by columns and sets their parent to the given alias.
// Useful for grouping joined table columns in nested mappings.
func GroupByWithParent(alias string, col expr.ColumnsExpr) bob.Mod[*dialect.SelectQuery] {
	return sm.GroupBy(col.WithParent(alias).DisableAlias())
}

// Scan is a helper function that creates a StructMapper with NullTypeConverter, so that NULL values are skipped during scanning.
// Example usage: bob.All(ctx, exec, psql.Select(query...), bob_helpers.Scan[BottomUpRow]())
func Scan[T any](opts ...scan.MappingOption) scan.Mapper[T] {
	allOpts := append([]scan.MappingOption{scan.WithTypeConverter(nullTypeConverter{})}, opts...)
	return scan.StructMapper[T](allOpts...)
}

// nullTypeConverter is a custom TypeConverter that skips NULL values during scanning even if the destination type does not support NULLs.
// Example usage:  psql.Select(query...), scan.StructMapper[MyStruct](scan.WithTypeConverter(bob_helpers.NullTypeConverter{}))
// This was stolen from https://github.com/stephenafamo/bob/blame/96da65fd88a50ae532079e8ea69746183f4af3a1/orm/load.go#L380
type nullTypeConverter struct{}

type wrapper struct {
	IsNull bool
	V      any
}

// Scan implements the sql.Scanner interface. If the wrapped type implements
// sql.Scanner then it will call that.
func (v *wrapper) Scan(value any) error {
	if value == nil {
		v.IsNull = true
		return nil
	}

	if scanner, ok := v.V.(sql.Scanner); ok {
		return scanner.Scan(value)
	}

	return opt.ConvertAssign(v.V, value)
}

func (nullTypeConverter) TypeToDestination(typ reflect.Type) reflect.Value {
	val := reflect.ValueOf(&wrapper{
		V: reflect.New(typ).Interface(),
	})

	return val
}

func (nullTypeConverter) ValueFromDestination(val reflect.Value) reflect.Value {
	return val.Elem().FieldByName("V").Elem().Elem()
}
