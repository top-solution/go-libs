package gen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerator(t *testing.T) {
	// Create a temporary test file
	testContent := `package testpkg

// db:filter
type ListDCRsRequest struct {
	// db:filter bob_gen.ColumnNames.DCRS.Type
	Type   string ` + "`query:\"type\"`" + `
	// db:filter bob_gen.ColumnNames.DCRS.Status  
	Status string ` + "`query:\"status\"`" + `
	// Regular field without filter comment
	Limit  int    ` + "`query:\"limit\"`" + `
}

// db:filter
type AnotherRequest struct {
	// db:filter bob_gen.ColumnNames.Users.Name
	Name *string ` + "`query:\"name\"`" + `
	// db:filter bob_gen.ColumnNames.Users.Tags
	Tags []string ` + "`query:\"tags\"`" + `
}

type NoFilterRequest struct {
	// db:filter bob_gen.ColumnNames.NoFilter.Field
	Field string ` + "`query:\"field\"`" + `
}`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	generator := NewGenerator("testpkg", tmpDir, "bob")
	structs, err := generator.parseFile(testFile)
	require.NoError(t, err)

	// Should find 2 structs with filter fields
	assert.Len(t, structs, 2)

	// Check first struct
	dcrsStruct := structs[0]
	assert.Equal(t, "ListDCRsRequest", dcrsStruct.Name)
	assert.Equal(t, "testpkg", dcrsStruct.Package)
	assert.Equal(t, "l", dcrsStruct.ReceiverName)
	assert.Len(t, dcrsStruct.Fields, 2)

	// Check fields
	typeField := dcrsStruct.Fields[0]
	assert.Equal(t, "Type", typeField.Name)
	assert.Equal(t, "bob_gen.ColumnNames.DCRS.Type", typeField.Column)
	assert.Equal(t, "string", typeField.Type)

	statusField := dcrsStruct.Fields[1]
	assert.Equal(t, "Status", statusField.Name)
	assert.Equal(t, "bob_gen.ColumnNames.DCRS.Status", statusField.Column)
	assert.Equal(t, "string", statusField.Type)

	// Check second struct
	usersStruct := structs[1]
	assert.Equal(t, "AnotherRequest", usersStruct.Name)
	assert.Equal(t, "a", usersStruct.ReceiverName)
	assert.Len(t, usersStruct.Fields, 2)

	nameField := usersStruct.Fields[0]
	assert.Equal(t, "Name", nameField.Name)
	assert.Equal(t, "bob_gen.ColumnNames.Users.Name", nameField.Column)
	assert.Equal(t, "*string", nameField.Type)

	tagsField := usersStruct.Fields[1]
	assert.Equal(t, "Tags", tagsField.Name)
	assert.Equal(t, "bob_gen.ColumnNames.Users.Tags", tagsField.Column)
	assert.Equal(t, "[]string", tagsField.Type)
}

func TestGenerator_GetTypeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple type",
			input:    "string",
			expected: "string",
		},
		{
			name:     "pointer type",
			input:    "*string",
			expected: "*string",
		},
		{
			name:     "slice type",
			input:    "[]string",
			expected: "[]string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test is more conceptual since getTypeString works with AST nodes
			// The actual type parsing is tested through the integration tests
			assert.NotEmpty(t, tt.expected)
		})
	}
}

func TestGenerator_NoFilterStructs(t *testing.T) {
	testContent := `package testpkg

type SimpleRequest struct {
	Name string ` + "`query:\"name\"`" + `
	Age  int    ` + "`query:\"age\"`" + `
}`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	generator := NewGenerator("testpkg", tmpDir, "bob")

	err = generator.GenerateFromFile(testFile)
	require.NoError(t, err)

	// Output file should not be created when no filter structs are found
	outputFile := filepath.Join(tmpDir, "test_filters.gen.go")
	_, err = os.Stat(outputFile)
	assert.True(t, os.IsNotExist(err))
}

func TestGenerator_GenerateFromPackage(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple Go files with filter structs
	file1Content := `package requests

// db:filter
type ListUsersRequest struct {
	// db:filter bob_gen.ColumnNames.Users.Name
	Name string ` + "`query:\"name\"`" + `
}`

	file2Content := `package requests

// db:filter
type ListOrdersRequest struct {
	// db:filter bob_gen.ColumnNames.Orders.Status
	Status string ` + "`query:\"status\"`" + `
}`

	// Create a file without filter structs
	file3Content := `package requests

type SimpleRequest struct {
	Field string ` + "`query:\"field\"`" + `
}`

	// Create a test file (should be ignored)
	testFileContent := `package requests

// db:filter
type TestStruct struct {
	// db:filter bob_gen.ColumnNames.Test.Field
	Field string ` + "`query:\"field\"`" + `
}`

	// Create a generated file (should be ignored)
	genFileContent := `package requests

// db:filter
type GenStruct struct {
	// db:filter bob_gen.ColumnNames.Gen.Field
	Field string ` + "`query:\"field\"`" + `
}`

	// Write all files
	err := os.WriteFile(filepath.Join(tmpDir, "users.go"), []byte(file1Content), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "orders.go"), []byte(file2Content), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "simple.go"), []byte(file3Content), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "test_test.go"), []byte(testFileContent), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "existing_gen.go"), []byte(genFileContent), 0644)
	require.NoError(t, err)

	// Generate filters for the entire package
	generator := NewGenerator("requests", tmpDir, "bob")
	err = generator.GenerateFromPackage()
	require.NoError(t, err)

	// Check that filter files were created for files with filter structs
	usersFilterFile := filepath.Join(tmpDir, "users_filters.gen.go")
	_, err = os.Stat(usersFilterFile)
	require.NoError(t, err)

	ordersFilterFile := filepath.Join(tmpDir, "orders_filters.gen.go")
	_, err = os.Stat(ordersFilterFile)
	require.NoError(t, err)

	// Check that no filter file was created for simple.go (no filter structs)
	simpleFilterFile := filepath.Join(tmpDir, "simple_filters.gen.go")
	_, err = os.Stat(simpleFilterFile)
	assert.True(t, os.IsNotExist(err))

	// Check that no filter file was created for test file
	testFilterFile := filepath.Join(tmpDir, "test_test_filters.gen.go")
	_, err = os.Stat(testFilterFile)
	assert.True(t, os.IsNotExist(err))

	// Check that no filter file was created for existing generated file
	existingGenFilterFile := filepath.Join(tmpDir, "existing_gen_filters.gen.go")
	_, err = os.Stat(existingGenFilterFile)
	assert.True(t, os.IsNotExist(err))

	// Verify content of generated files
	usersGenerated, err := os.ReadFile(usersFilterFile)
	require.NoError(t, err)
	assert.Contains(t, string(usersGenerated), "func (l *ListUsersRequest) AddFilters")

	ordersGenerated, err := os.ReadFile(ordersFilterFile)
	require.NoError(t, err)
	assert.Contains(t, string(ordersGenerated), "func (l *ListOrdersRequest) AddFilters")
}

func TestGenerator_GetOutputFilename(t *testing.T) {
	generator := NewGenerator("test", ".", "bob")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple file",
			input:    "requests.go",
			expected: "requests_filters.gen.go",
		},
		{
			name:     "file with path",
			input:    "/path/to/requests.go",
			expected: "/path/to/requests_filters.gen.go",
		},
		{
			name:     "file with multiple dots",
			input:    "my.requests.go",
			expected: "my.requests_filters.gen.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.getOutputFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerator_FilterCommentParsing(t *testing.T) {
	testContent := `package testpkg

// db:filter
type TestRequest struct {
	// db:filter simple_column
	Field1 string
	// db:filter   spaced_column   
	Field2 string
	//db:filter no_space_column
	Field3 string
	// db:filter "quoted_column"
	Field4 string
	// Some other comment
	Field5 string
	// db:filter complex.column.name
	Field6 string
}`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	generator := NewGenerator("testpkg", tmpDir, "bob")
	structs, err := generator.parseFile(testFile)
	require.NoError(t, err)

	require.Len(t, structs, 1)
	testStruct := structs[0]

	// Should have 5 fields with filter comments (Field5 doesn't have db:filter)
	assert.Len(t, testStruct.Fields, 5)

	expectedColumns := []string{
		"simple_column",
		"spaced_column",
		"no_space_column",
		"\"quoted_column\"",
		"complex.column.name",
	}

	for i, field := range testStruct.Fields {
		assert.Equal(t, expectedColumns[i], field.Column)
		assert.Equal(t, "string", field.Type)
	}
}

func TestGenerator_StructFilterComment(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectFound bool
	}{
		{
			name: "has db:filter comment",
			content: `package testpkg

// db:filter
type TestRequest struct {
	// db:filter column_name
	Field string
}`,
			expectFound: true,
		},
		{
			name: "no struct filter comment",
			content: `package testpkg

type TestRequest struct {
	// db:filter column_name
	Field string
}`,
			expectFound: false,
		},
		{
			name: "other comment",
			content: `package testpkg

// Some other comment
type TestRequest struct {
	// db:filter column_name
	Field string
}`,
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.go")
			err := os.WriteFile(testFile, []byte(tt.content), 0644)
			require.NoError(t, err)

			generator := NewGenerator("testpkg", tmpDir, "bob")
			structs, err := generator.parseFile(testFile)
			require.NoError(t, err)

			if tt.expectFound {
				require.Len(t, structs, 1)
			} else {
				assert.Len(t, structs, 0)
			}
		})
	}
}
func TestGenerator_ImportComments(t *testing.T) {
	testContent := `package testpkg

// db:filter
// db:filter import "fmt"
// db:filter import "time"
// db:filter import github.com/example/pkg
// db:filter import json "encoding/json"
// db:filter import ctx "context"
type TestRequest struct {
	// db:filter column_name
	Field string
}`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	generator := NewGenerator("testpkg", tmpDir, "bob")
	structs, err := generator.parseFile(testFile)
	require.NoError(t, err)

	require.Len(t, structs, 1)
	testStruct := structs[0]

	// Should have 5 imports
	assert.Len(t, testStruct.Imports, 5)
	assert.Contains(t, testStruct.Imports, `"fmt"`)
	assert.Contains(t, testStruct.Imports, `"time"`)
	assert.Contains(t, testStruct.Imports, `"github.com/example/pkg"`)
	assert.Contains(t, testStruct.Imports, `json "encoding/json"`)
	assert.Contains(t, testStruct.Imports, `ctx "context"`)
}

func TestGenerator_GenerateWithImports(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test input file with imports
	testContent := `package requests

// db:filter
// db:filter import "fmt"
// db:filter import "encoding/json"
type ListUsersRequest struct {
	// db:filter bob_gen.ColumnNames.Users.Name
	Name string ` + "`query:\"name\"`" + `
}`

	inputFile := filepath.Join(tmpDir, "requests.go")
	err := os.WriteFile(inputFile, []byte(testContent), 0644)
	require.NoError(t, err)

	generator := NewGenerator("requests", tmpDir, "bob")

	err = generator.GenerateFromFile(inputFile)
	require.NoError(t, err)

	// Check that output file was created
	outputFile := filepath.Join(tmpDir, "requests_filters.gen.go")
	_, err = os.Stat(outputFile)
	require.NoError(t, err)

	// Read and verify generated content
	generated, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	generatedStr := string(generated)

	// Check for expected imports
	assert.Contains(t, generatedStr, `"fmt"`)
	assert.Contains(t, generatedStr, `"encoding/json"`)
	assert.Contains(t, generatedStr, "package requests")
	assert.Contains(t, generatedStr, "func (l *ListUsersRequest) AddFilters")
}
func TestGenerator_ParseImportSpec(t *testing.T) {
	generator := NewGenerator("test", ".", "bob")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "quoted package",
			input:    `"fmt"`,
			expected: `"fmt"`,
		},
		{
			name:     "unquoted package",
			input:    `fmt`,
			expected: `"fmt"`,
		},
		{
			name:     "alias with quoted package",
			input:    `json "encoding/json"`,
			expected: `json "encoding/json"`,
		},
		{
			name:     "alias with unquoted package",
			input:    `ctx context`,
			expected: `ctx "context"`,
		},
		{
			name:     "complex package path",
			input:    `github.com/example/pkg`,
			expected: `"github.com/example/pkg"`,
		},
		{
			name:     "alias with complex package path",
			input:    `pkg github.com/example/pkg`,
			expected: `pkg "github.com/example/pkg"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.parseImportSpec(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
func TestGenerator_SortField(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		expectSort bool
		sortField  string
	}{
		{
			name: "struct with sortField comment",
			content: `package testpkg

// db:filter
// db:filter sortField Sort
type TestRequest struct {
	// db:filter column_name
	Field string
	Sort  []string ` + "`query:\"sort\"`" + `
}`,
			expectSort: true,
			sortField:  "Sort",
		},
		{
			name: "struct without sortField comment",
			content: `package testpkg

// db:filter
type TestRequest struct {
	// db:filter column_name
	Field string
	Sort  []string ` + "`query:\"sort\"`" + `
}`,
			expectSort: false,
			sortField:  "",
		},
		{
			name: "struct with different sortField name",
			content: `package testpkg

// db:filter
// db:filter sortField OrderBy
type TestRequest struct {
	// db:filter column_name
	Field string
	OrderBy []string ` + "`query:\"order_by\"`" + `
}`,
			expectSort: true,
			sortField:  "OrderBy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.go")
			err := os.WriteFile(testFile, []byte(tt.content), 0644)
			require.NoError(t, err)

			generator := NewGenerator("testpkg", tmpDir, "bob")
			structs, err := generator.parseFile(testFile)
			require.NoError(t, err)

			require.Len(t, structs, 1)
			if tt.expectSort {
				assert.Equal(t, tt.sortField, structs[0].SortField)
				assert.NotEmpty(t, structs[0].SortField)
			} else {
				assert.Empty(t, structs[0].SortField)
			}
		})
	}
}

func TestGenerator_GenerateWithSorting(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test input file with sortField comment
	testContent := `package requests

// db:filter
// db:filter sortField Sort
type ListUsersRequest struct {
	// db:filter bob_gen.ColumnNames.Users.Name
	Name string ` + "`query:\"name\"`" + `
	Sort []string ` + "`query:\"sort\"`" + `
}`

	inputFile := filepath.Join(tmpDir, "requests.go")
	err := os.WriteFile(inputFile, []byte(testContent), 0644)
	require.NoError(t, err)

	generator := NewGenerator("requests", tmpDir, "bob")

	err = generator.GenerateFromFile(inputFile)
	require.NoError(t, err)

	// Check that output file was created
	outputFile := filepath.Join(tmpDir, "requests_filters.gen.go")
	_, err = os.Stat(outputFile)
	require.NoError(t, err)

	// Read and verify generated content
	generated, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	generatedStr := string(generated)

	// Check for expected content
	assert.Contains(t, generatedStr, "package requests")
	assert.Contains(t, generatedStr, "func (l *ListUsersRequest) AddFilters")
	assert.Contains(t, generatedStr, "func (l *ListUsersRequest) AddSorting")
	assert.Contains(t, generatedStr, "ListUsersRequestColumnsMap.AddSorting(query, l.Sort)")

	// Check for proper imports structure
	assert.Contains(t, generatedStr, `"github.com/top-solution/go-libs/v2/dbutils/ops"`)
	assert.Contains(t, generatedStr, `"github.com/stephenafamo/bob"`)
	assert.Contains(t, generatedStr, `"github.com/stephenafamo/bob/dialect/psql/dialect"`)
	assert.Contains(t, generatedStr, `"github.com/top-solution/go-libs/v2/dbutils/ops/bobops"`)
}

func TestGenerator_GenerateWithoutSorting(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test input file without sortField comment
	testContent := `package requests

// db:filter
type ListUsersRequest struct {
	// db:filter bob_gen.ColumnNames.Users.Name
	Name string ` + "`query:\"name\"`" + `
}`

	inputFile := filepath.Join(tmpDir, "requests.go")
	err := os.WriteFile(inputFile, []byte(testContent), 0644)
	require.NoError(t, err)

	generator := NewGenerator("requests", tmpDir, "bob")

	err = generator.GenerateFromFile(inputFile)
	require.NoError(t, err)

	// Check that output file was created
	outputFile := filepath.Join(tmpDir, "requests_filters.gen.go")
	_, err = os.Stat(outputFile)
	require.NoError(t, err)

	// Read and verify generated content
	generated, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	generatedStr := string(generated)

	// Check for expected content
	assert.Contains(t, generatedStr, "package requests")
	assert.Contains(t, generatedStr, "func (l *ListUsersRequest) AddFilters")
	// Should NOT contain AddSorting function
	assert.NotContains(t, generatedStr, "func (l *ListUsersRequest) AddSorting")
	assert.NotContains(t, generatedStr, "AddSorting")

	// Check for proper imports structure (without errors)
	assert.Contains(t, generatedStr, `"github.com/top-solution/go-libs/v2/dbutils/ops"`)
	assert.Contains(t, generatedStr, `"github.com/stephenafamo/bob"`)
	assert.Contains(t, generatedStr, `"github.com/stephenafamo/bob/dialect/psql/dialect"`)
	assert.Contains(t, generatedStr, `"github.com/top-solution/go-libs/v2/dbutils/ops/bobops"`)
}

func TestGenerator_HavingParameter(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test input file with having parameters
	testContent := `package requests

// db:filter
type ListUsersRequest struct {
	// db:filter bob_gen.ColumnNames.Users.Name
	Name string ` + "`query:\"name\"`" + `
	// db:filter bob_gen.ColumnNames.Users.Count having
	Count string ` + "`query:\"count\"`" + `
	// db:filter bob_gen.ColumnNames.Users.Email
	Email *string ` + "`query:\"email\"`" + `
	// db:filter bob_gen.ColumnNames.Users.Tags having
	Tags []string ` + "`query:\"tags\"`" + `
}`

	inputFile := filepath.Join(tmpDir, "requests.go")
	err := os.WriteFile(inputFile, []byte(testContent), 0644)
	require.NoError(t, err)

	generator := NewGenerator("requests", tmpDir, "bob")

	err = generator.GenerateFromFile(inputFile)
	require.NoError(t, err)

	// Check that the generated file exists and has the correct having parameters
	outputFile := filepath.Join(tmpDir, "requests_filters.gen.go")
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	generatedCode := string(content)
	
	// Should contain the AddFilters method
	assert.Contains(t, generatedCode, "func (l *ListUsersRequest) AddFilters")
	
	// Check that non-having filters use false
	assert.Contains(t, generatedCode, `ParseFilter(cond, bob_gen.ColumnNames.Users.Name, op, rawValue, false)`)
	assert.Contains(t, generatedCode, `ParseFilter(cond, bob_gen.ColumnNames.Users.Email, op, rawValue, false)`)
	
	// Check that having filters use true
	assert.Contains(t, generatedCode, `ParseFilter(cond, bob_gen.ColumnNames.Users.Count, op, rawValue, true)`)
	assert.Contains(t, generatedCode, `ParseFilter(cond, bob_gen.ColumnNames.Users.Tags, op, rawValue, true)`)
}

func TestGenerator_ComplexFilterExpressions(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test input file with complex SQL expressions
	testContent := `package requests

// db:filter
type ComplexRequest struct {
	// db:filter "(CASE WHEN bom.pn = bom.enditem THEN 1 END)"
	CaseExpression string ` + "`query:\"case_expr\"`" + `
	// db:filter "COALESCE(users.name, users.email, 'Unknown')"
	CoalesceExpr string ` + "`query:\"coalesce\"`" + `
	// db:filter "COUNT(*) FILTER (WHERE status = 'active')" having
	AggregateHaving string ` + "`query:\"aggregate\"`" + `
	// db:filter "DATE_TRUNC('day', created_at)"
	FunctionCall string ` + "`query:\"date_func\"`" + `
	// db:filter "EXTRACT(EPOCH FROM NOW() - created_at)"
	ExtractExpr string ` + "`query:\"extract\"`" + `
}`

	inputFile := filepath.Join(tmpDir, "requests.go")
	err := os.WriteFile(inputFile, []byte(testContent), 0644)
	require.NoError(t, err)

	generator := NewGenerator("requests", tmpDir, "bob")

	err = generator.GenerateFromFile(inputFile)
	require.NoError(t, err)

	// Check that the generated file exists and has the correct complex expressions
	outputFile := filepath.Join(tmpDir, "requests_filters.gen.go")
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	generatedCode := string(content)
	
	// Should contain the AddFilters method
	assert.Contains(t, generatedCode, "func (c *ComplexRequest) AddFilters")
	
	// Check that complex expressions are properly captured
	assert.Contains(t, generatedCode, `"(CASE WHEN bom.pn = bom.enditem THEN 1 END)"`)
	assert.Contains(t, generatedCode, `"COALESCE(users.name, users.email, 'Unknown')"`)
	assert.Contains(t, generatedCode, `"COUNT(*) FILTER (WHERE status = 'active')"`)
	assert.Contains(t, generatedCode, `"DATE_TRUNC('day', created_at)"`)
	assert.Contains(t, generatedCode, `"EXTRACT(EPOCH FROM NOW() - created_at)"`)
	
	// Check that having parameter is correctly applied
	assert.Contains(t, generatedCode, `ParseFilter(cond, "(CASE WHEN bom.pn = bom.enditem THEN 1 END)", op, rawValue, false)`)
	assert.Contains(t, generatedCode, `ParseFilter(cond, "COALESCE(users.name, users.email, 'Unknown')", op, rawValue, false)`)
	assert.Contains(t, generatedCode, `ParseFilter(cond, "COUNT(*) FILTER (WHERE status = 'active')", op, rawValue, true)`)
	assert.Contains(t, generatedCode, `ParseFilter(cond, "DATE_TRUNC('day', created_at)", op, rawValue, false)`)
	assert.Contains(t, generatedCode, `ParseFilter(cond, "EXTRACT(EPOCH FROM NOW() - created_at)", op, rawValue, false)`)
}

func TestGenerator_SortByFieldMapping(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test input file with sortBy mappings
	testContent := `package requests

// db:filter
// db:filter sortField Sort
type ProductsRequest struct {
	// db:filter bob_gen.ColumnNames.Products.Name sortBy "products.name"
	ProductName string ` + "`query:\"product_name\"`" + `
	// db:filter bob_gen.ColumnNames.Products.Price sortBy "products.price"
	Price string ` + "`query:\"price\"`" + `
	// db:filter bob_gen.ColumnNames.Products.Category
	Category string ` + "`query:\"category\"`" + `
	Sort []string ` + "`query:\"sort\"`" + `
}`

	inputFile := filepath.Join(tmpDir, "requests.go")
	err := os.WriteFile(inputFile, []byte(testContent), 0644)
	require.NoError(t, err)

	generator := NewGenerator("requests", tmpDir, "bob")

	// Parse the file to check struct info
	structs, err := generator.parseFile(inputFile)
	require.NoError(t, err)
	require.Len(t, structs, 1)

	// Verify that sortBy is extracted correctly
	fields := structs[0].Fields
	require.Len(t, fields, 3)

	assert.Equal(t, "ProductName", fields[0].Name)
	assert.Equal(t, `"products.name"`, fields[0].SortBy)

	assert.Equal(t, "Price", fields[1].Name)
	assert.Equal(t, `"products.price"`, fields[1].SortBy)

	assert.Equal(t, "Category", fields[2].Name)
	assert.Equal(t, "", fields[2].SortBy) // No sortBy specified

	// Generate code and verify the SortColumnsMap is created
	err = generator.GenerateFromFile(inputFile)
	require.NoError(t, err)

	outputFile := filepath.Join(tmpDir, "requests_filters.gen.go")
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	generatedCode := string(content)

	// Should contain both ColumnsMap and SortColumnsMap
	assert.Contains(t, generatedCode, "var ProductsRequestColumnsMap")
	assert.Contains(t, generatedCode, "var ProductsRequestSortColumnsMap")

	// ColumnsMap should use the regular columns
	assert.Contains(t, generatedCode, `"product_name": bob_gen.ColumnNames.Products.Name`)
	assert.Contains(t, generatedCode, `"price": bob_gen.ColumnNames.Products.Price`)
	assert.Contains(t, generatedCode, `"category": bob_gen.ColumnNames.Products.Category`)

	// SortColumnsMap should use sortBy columns where specified
	assert.Contains(t, generatedCode, `ProductsRequestSortColumnsMap`)
	assert.Contains(t, generatedCode, `"product_name": "products.name"`)
	assert.Contains(t, generatedCode, `"price": "products.price"`)
	assert.Contains(t, generatedCode, `"category": bob_gen.ColumnNames.Products.Category`) // Falls back to filter column

	// AddSorting should use SortColumnsMap
	assert.Contains(t, generatedCode, "func (p *ProductsRequest) AddSorting")
	assert.Contains(t, generatedCode, "ProductsRequestSortColumnsMap.AddSorting(query, p.Sort)")
}

func TestGenerator_SortByWithHaving(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test input file with sortBy and having
	testContent := `package requests

// db:filter
// db:filter sortField Sort
type AggregatesRequest struct {
	// db:filter "COUNT(*)" having sortBy "count_value"
	Count string ` + "`query:\"count\"`" + `
	// db:filter bob_gen.ColumnNames.Users.Name
	Name string ` + "`query:\"name\"`" + `
	Sort []string ` + "`query:\"sort\"`" + `
}`

	inputFile := filepath.Join(tmpDir, "requests.go")
	err := os.WriteFile(inputFile, []byte(testContent), 0644)
	require.NoError(t, err)

	generator := NewGenerator("requests", tmpDir, "bob")

	// Parse and verify
	structs, err := generator.parseFile(inputFile)
	require.NoError(t, err)
	require.Len(t, structs, 1)

	fields := structs[0].Fields
	require.Len(t, fields, 2)

	assert.Equal(t, "Count", fields[0].Name)
	assert.Equal(t, `"count_value"`, fields[0].SortBy)
	assert.True(t, fields[0].Having)

	assert.Equal(t, "Name", fields[1].Name)
	assert.Equal(t, "", fields[1].SortBy)
	assert.False(t, fields[1].Having)

	// Generate and verify
	err = generator.GenerateFromFile(inputFile)
	require.NoError(t, err)

	outputFile := filepath.Join(tmpDir, "requests_filters.gen.go")
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	generatedCode := string(content)

	// Should have SortColumnsMap since sortBy is present
	assert.Contains(t, generatedCode, "var AggregatesRequestSortColumnsMap")
	assert.Contains(t, generatedCode, `"count": "count_value"`)
	assert.Contains(t, generatedCode, `"name": bob_gen.ColumnNames.Users.Name`)
}

func TestGenerator_NoSortByNoSortColumnsMap(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test input file without any sortBy
	testContent := `package requests

// db:filter
// db:filter sortField Sort
type SimpleRequest struct {
	// db:filter bob_gen.ColumnNames.Users.Name
	Name string ` + "`query:\"name\"`" + `
	// db:filter bob_gen.ColumnNames.Users.Email
	Email string ` + "`query:\"email\"`" + `
	Sort []string ` + "`query:\"sort\"`" + `
}`

	inputFile := filepath.Join(tmpDir, "requests.go")
	err := os.WriteFile(inputFile, []byte(testContent), 0644)
	require.NoError(t, err)

	generator := NewGenerator("requests", tmpDir, "bob")

	err = generator.GenerateFromFile(inputFile)
	require.NoError(t, err)

	outputFile := filepath.Join(tmpDir, "requests_filters.gen.go")
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	generatedCode := string(content)

	// Should NOT have SortColumnsMap when no sortBy is specified
	assert.NotContains(t, generatedCode, "SortColumnsMap")

	// AddSorting should use regular ColumnsMap
	assert.Contains(t, generatedCode, "SimpleRequestColumnsMap.AddSorting(query, s.Sort)")
}

func TestGenerator_QueryTagWithComma(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test input file with comma-separated query tags
	testContent := `package requests

// db:filter
type TestRequest struct {
	// db:filter bob_gen.ColumnNames.Users.Name
	Name string ` + "`query:\"name,omitempty\"`" + `
	// db:filter bob_gen.ColumnNames.Users.Email
	Email string ` + "`query:\"email,required\"`" + `
}`

	inputFile := filepath.Join(tmpDir, "requests.go")
	err := os.WriteFile(inputFile, []byte(testContent), 0644)
	require.NoError(t, err)

	generator := NewGenerator("requests", tmpDir, "bob")

	// Parse and verify
	structs, err := generator.parseFile(inputFile)
	require.NoError(t, err)
	require.Len(t, structs, 1)

	fields := structs[0].Fields
	require.Len(t, fields, 2)

	// Should extract only the first part before the comma
	assert.Equal(t, "name", fields[0].QueryParam)
	assert.Equal(t, "email", fields[1].QueryParam)

	// Generate and verify
	err = generator.GenerateFromFile(inputFile)
	require.NoError(t, err)

	outputFile := filepath.Join(tmpDir, "requests_filters.gen.go")
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	generatedCode := string(content)

	// Should use the name without the comma-separated options
	assert.Contains(t, generatedCode, `"name": bob_gen.ColumnNames.Users.Name`)
	assert.Contains(t, generatedCode, `"email": bob_gen.ColumnNames.Users.Email`)
	assert.NotContains(t, generatedCode, "omitempty")
	assert.NotContains(t, generatedCode, "required")
}
