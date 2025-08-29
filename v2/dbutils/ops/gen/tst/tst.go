//go:generate go run ../cmd/main.go bob .
package tst

type Sortable struct {
	Sort []string
}

// db:filter
// db:filter import "fmt"
// db:filter sortField Sort
type TestStruct struct {
	Sortable
	// db:filter "stuff"
	Test string `query:"test"`
	// db:filter fmt.Sprintf("heee")
	Test2 *string `query:"test2"`
	// db:filter "EEEI"
	Test3 []string `query:"test3"`
	// db:filter "group_col" having
	Test4 string `query:"test4"`
	// db:filter "having_ptr_col" having
	Test5 *string `query:"test5"`
	// db:filter "having_array_col" having
	Test6 []string `query:"test6"`
}
