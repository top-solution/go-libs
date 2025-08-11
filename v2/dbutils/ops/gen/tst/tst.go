//go:generate go run ../cmd/main.go bob .
package tst

// db:filter
// db:filter import "fmt"
type TestStruct struct {
	// db:filter "stuff"
	Test string `query:"test"`
	// db:filter fmt.Sprintf("heee")
	Test2 *string `query:"test2"`
	// db:filter "EEEI"
	Test3 []string `query:"test3"`
	Sort  []string `query:"sort"`
}
