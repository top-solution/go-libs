//go:generate go run ../cmd/main.go bob .
package tst

type Sortable struct {
	Sort []string
}

var simple_column = "simple_column"

var tablename = struct {
	column_name string
}{
	column_name: "test_column",
}

// db:filter
// db:filter import "fmt"
// db:filter sortField Sort
type TestStruct struct {
	Sortable
	// db:filter "stuff" sortBy "sorted_stuff"
	Test string `query:"test"`
	// db:filter fmt.Sprintf("heee") sortBy fmt.Sprintf("sorted_heee")
	Test2 *string `query:"test2"`
	// db:filter "EEEI"
	Test3 []string `query:"test3"`
	// db:filter "group_col" having
	Test4 string `query:"test4"`
	// db:filter "having_ptr_col" having
	Test5 *string `query:"test5"`
	// db:filter "having_array_col" having
	Test6 []string `query:"test6"`
	// db:filter "(CASE WHEN bom.pn = bom.enditem THEN 1 END)"
	Test7 string `query:"test7"`
	// db:filter "COALESCE(users.name, users.email, 'Unknown')"
	Test8 string `query:"test8"`
	// db:filter "COUNT(*) FILTER (WHERE status = 'active')" having
	Test9 string `query:"test9"`
	// db:filter "DATE_TRUNC('day', created_at)"
	Test10 string `query:"test10"`
	// db:filter simple_column
	Test11 string `query:"test11"`
	// db:filter tablename.column_name having
	Test12 string `query:"test12"`
}
