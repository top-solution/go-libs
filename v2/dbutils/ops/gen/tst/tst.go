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
}
