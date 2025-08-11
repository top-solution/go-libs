package bob_helpers

import (
	"database/sql"
	"reflect"
	"time"
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
