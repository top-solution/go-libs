package bob_helpers

import (
	"database/sql"
	"reflect"
	"time"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
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

// TableAliasToPrefix returns a properly formatted prefix string for use in
// bob-generated SQL queries.
//
// In bob, you can use `--prefix:` annotations in your SQL files to map
// columns to a Go struct that uses a specific table alias. This function
// helps generate the correct prefix format (`"<alias>."`) expected by bob
// for that mapping.
//
// Example usage:
//
//	prefix := TableAliasToPrefix("posts")
//	// prefix == "posts."
//
// In SQL (used by bob code generation):
//
//	--prefix:posts.
//	posts.id, posts.title
//
// This allows bob to correctly bind the selected columns to fields in a Go
// struct representing the `posts` table.
//
// Parameters:
//   - alias: the alias or table name to use as prefix.
//
// Returns:
//   - A string in the format "<alias>." if alias is not empty,
//     or an empty string otherwise.
func TableAliasToPrefix(alias string) string {
	return alias + "."
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
