package bob_helpers

import (
	"database/sql"
	"reflect"
	"time"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/expr"
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
