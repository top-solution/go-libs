//go:generate go run ../cmd/main.go bob .
package tst

// db:filter
type TestStruct struct {
	// db:filter "stuff"
	Test string `query:"test"`
	// db:filter "EEE"
	Test2 *string `query:"test2"`
	// db:filter "EEEI"
	Test3 []string `query:"test3"`
}
