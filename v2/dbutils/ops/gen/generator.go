package gen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

// FilterField represents a field that should have filter generation
type FilterField struct {
	Name       string // Field name in the struct
	Column     string // Database column name from comment
	QueryParam string // Query parameter name from struct tag
	Type       string // Field type
}

// StructInfo contains information about a struct that needs filter generation
type StructInfo struct {
	Name         string
	Package      string
	Fields       []FilterField
	ReceiverName string
	Imports      []string // Additional imports specified in comments
	SortField    string   // The field to sort by, if specified
}

// Generator handles the code generation for filter methods
type Generator struct {
	packageName string
	packageDir  string
	filterType  string
}

// NewGenerator creates a new generator instance
func NewGenerator(packageName, packageDir, filterType string) *Generator {
	return &Generator{
		packageName: packageName,
		packageDir:  packageDir,
		filterType:  filterType,
	}
}

// GenerateFromPackage scans all Go files in the package directory and generates filter methods
func (g *Generator) GenerateFromPackage() error {
	files, err := filepath.Glob(filepath.Join(g.packageDir, "*.go"))
	if err != nil {
		return fmt.Errorf("failed to find Go files: %w", err)
	}

	for _, file := range files {
		// Skip generated files and test files
		if strings.HasSuffix(file, "_test.go") || strings.Contains(file, "_gen.go") {
			continue
		}

		if err := g.generateFromFile(file); err != nil {
			return fmt.Errorf("failed to process file %s: %w", file, err)
		}
	}

	return nil
}

// GenerateFromFile parses a Go file and generates filter methods for structs with db:filter comments
func (g *Generator) GenerateFromFile(filename string) error {
	return g.generateFromFile(filename)
}

// generateFromFile is the internal implementation for processing a single file
func (g *Generator) generateFromFile(filename string) error {
	structs, err := g.parseFile(filename)
	if err != nil {
		return fmt.Errorf("failed to parse file %s: %w", filename, err)
	}

	if len(structs) == 0 {
		return nil // No structs with filter comments found
	}

	var structNames []string
	for _, s := range structs {
		structNames = append(structNames, s.Name)
	}

	fmt.Printf("Processing file %s (%v)\n", filename, strings.Join(structNames, ", "))

	// Generate output filename: file.go -> file_filters.gen.go
	outputFile := g.getOutputFilename(filename)
	return g.generateCode(structs, outputFile)
}

// getOutputFilename generates the output filename based on the input filename
func (g *Generator) getOutputFilename(inputFile string) string {
	dir := filepath.Dir(inputFile)
	base := filepath.Base(inputFile)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]
	return filepath.Join(dir, name+"_filters.gen.go")
}

// parseFile parses a Go file and extracts struct information
func (g *Generator) parseFile(filename string) ([]StructInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var structs []StructInfo

	// Create a map of positions to comments for easier lookup
	commentMap := make(map[token.Pos]*ast.CommentGroup)
	for _, cg := range node.Comments {
		commentMap[cg.Pos()] = cg
	}

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.GenDecl:
			if x.Tok == token.TYPE {
				for _, spec := range x.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if structType, ok := typeSpec.Type.(*ast.StructType); ok {
							// Check for struct-level db:filter comment in the GenDecl doc
							hasFilter, imports, sortField := g.parseFilterComments(x.Doc)
							if hasFilter {
								structInfo := g.parseStruct(typeSpec.Name.Name, structType)
								if len(structInfo.Fields) > 0 {
									structInfo.Package = node.Name.Name
									structInfo.Imports = imports
									structInfo.SortField = sortField
									structs = append(structs, structInfo)
								}
							}
						}
					}
				}
			}
		}
		return true
	})

	return structs, nil
}

// parseFilterComments checks if the struct has a db:filter comment and extracts imports
func (g *Generator) parseFilterComments(doc *ast.CommentGroup) (bool, []string, string) {
	if doc == nil {
		return false, nil, ""
	}

	structFilterRegex := regexp.MustCompile(`//\s*db:filter\s*$`)
	importRegex := regexp.MustCompile(`//\s*db:filter\s+import\s+(.+)`)
	sortRegex := regexp.MustCompile(`//\s*db:filter\s+sortField\s+(.+)`)

	hasFilter := false
	var imports []string
	var sortField string

	for _, comment := range doc.List {
		if structFilterRegex.MatchString(comment.Text) {
			hasFilter = true
		} else if matches := importRegex.FindStringSubmatch(comment.Text); len(matches) > 1 {
			importSpec := strings.TrimSpace(matches[1])
			imports = append(imports, g.parseImportSpec(importSpec))
		} else if matches := sortRegex.FindStringSubmatch(comment.Text); len(matches) > 1 {
			sortField = strings.TrimSpace(matches[1])
		}

	}

	return hasFilter, imports, sortField
}

// parseImportSpec parses import specifications with optional aliases
// Supports formats like:
// - "package"
// - package
// - alias "package"
// - alias package
func (g *Generator) parseImportSpec(spec string) string {
	spec = strings.TrimSpace(spec)

	// Check if it contains a space (indicating an alias)
	parts := strings.Fields(spec)

	if len(parts) == 1 {
		// Single part - just a package path
		pkg := strings.Trim(parts[0], `"`)
		return `"` + pkg + `"`
	} else if len(parts) == 2 {
		// Two parts - alias and package
		alias := parts[0]
		pkg := strings.Trim(parts[1], `"`)
		return alias + ` "` + pkg + `"`
	}

	// Fallback - return as is with quotes if not already quoted
	if !strings.HasPrefix(spec, `"`) {
		return `"` + spec + `"`
	}
	return spec
}

var filterCommentRegex = regexp.MustCompile(`//\s*db:filter\s+(.+)`)

// parseStruct extracts filter field information from a struct
func (g *Generator) parseStruct(name string, structType *ast.StructType) StructInfo {
	info := StructInfo{
		Name:         name,
		ReceiverName: strings.ToLower(name[:1]),
		Fields:       []FilterField{},
	}

	for _, field := range structType.Fields.List {
		if field.Doc == nil {
			continue
		}

		// Check for db:filter comment
		var column string
		for _, comment := range field.Doc.List {
			matches := filterCommentRegex.FindStringSubmatch(comment.Text)
			if len(matches) > 1 {
				column = strings.TrimSpace(matches[1])
				break
			}
		}

		if column == "" {
			continue
		}

		// Extract field information
		for _, fieldName := range field.Names {
			fieldType := g.getTypeString(field.Type)

			info.Fields = append(info.Fields, FilterField{
				Name:   fieldName.Name,
				Column: column,
				Type:   fieldType,
			})
		}
	}

	return info
}

// getTypeString converts an ast.Expr to a string representation
func (g *Generator) getTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + g.getTypeString(t.X)
	case *ast.SelectorExpr:
		return g.getTypeString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + g.getTypeString(t.Elt)
	default:
		return "interface{}"
	}
}

// generateCode generates the filter methods code
func (g *Generator) generateCode(structs []StructInfo, outputFile string) error {
	// Skip file creation if no structs
	if len(structs) == 0 {
		return nil
	}

	// Only generate for supported filter types
	if g.filterType != "bob" && g.filterType != "boiler" {
		return nil
	}

	tmpl := template.Must(template.New("filters").Parse(codeTemplate))

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Collect all unique additional imports from all structs
	importSet := make(map[string]bool)
	for _, s := range structs {
		for _, imp := range s.Imports {
			importSet[imp] = true
		}
	}

	var additionalImports []string
	for imp := range importSet {
		additionalImports = append(additionalImports, imp)
	}

	// Check if any struct has sorting
	hasSortingStructs := false
	for _, s := range structs {
		if s.SortField != "" {
			hasSortingStructs = true
			break
		}
	}

	data := struct {
		FilterType        string
		Package           string
		Structs           []StructInfo
		AdditionalImports []string
		HasSortingStructs bool
	}{
		FilterType:        g.filterType,
		Package:           g.packageName,
		Structs:           structs,
		AdditionalImports: additionalImports,
		HasSortingStructs: hasSortingStructs,
	}

	return tmpl.Execute(file, data)
}

const codeTemplate = `// Code generated by go-libs/v2/dbutils/ops/gen/cmd. DO NOT EDIT.

package {{.Package}}

import (
	{{if .HasSortingStructs}}"errors"{{end}}
	"github.com/top-solution/go-libs/v2/dbutils/ops"
	{{if eq .FilterType "bob"}}
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/top-solution/go-libs/v2/dbutils/ops/bobops"
	{{ else if eq .FilterType "boiler"}}
	"github.com/top-solution/go-libs/v2/dbutils/ops/boilerops"
    "github.com/volatiletech/sqlboiler/v4/queries/qm"
	{{end}}
{{range .AdditionalImports}}	{{.}}
{{end}}){{$lib := .FilterType}}
{{range .Structs}}{{$receiver := .ReceiverName}}

// AddFilters adds database filters based on the struct fields with db:filter comments
// DO NOT EDIT: This file is generated by go-libs/v2/dbutils/ops/gen/cmd
func ({{.ReceiverName}} *{{.Name}}) AddFilters(q {{if eq $lib "bob"}}*[]bob.Mod[*dialect.SelectQuery]{{else if eq $lib "boiler"}}*[]qm.QueryMod{{end}}) error {
	{{if eq $lib "bob"}}filterer := bobops.BobFilterer{}
	var qmods []bob.Mod[*dialect.SelectQuery]
	{{else if eq $lib "boiler"}}filterer := boilerops.BoilFilterer{}
	var qmods []qm.QueryMod{{end}}
	{{range .Fields}}{{if eq .Type "string"}}if {{$receiver}}.{{.Name}} != "" {
		op, cond, rawValue, err := ops.CurrentWhereFilters().Parse({{$receiver}}.{{.Name}})
		if err != nil {
			return err
		}

		qmod, _, _, err := filterer.ParseFilter(cond, {{.Column}}, op, rawValue, false)
		if err != nil {
			return err
		}
		qmods = append(qmods, qmod)
	}{{else if eq .Type "*string"}}
	if {{$receiver}}.{{.Name}} != nil && *{{$receiver}}.{{.Name}} != "" {
		op, cond, rawValue, err := ops.CurrentWhereFilters().Parse(*{{$receiver}}.{{.Name}})
		if err != nil {
			return err
		}

		qmod, _, _, err := filterer.ParseFilter(cond, {{.Column}}, op, rawValue, false)
		if err != nil {
			return err
		}
		qmods = append(qmods, qmod)
	}{{else if eq .Type "[]string"}}
	if len({{$receiver}}.{{.Name}}) > 0 {
	    for _, v := range {{$receiver}}.{{.Name}} {
			op, cond, rawValue, err := ops.CurrentWhereFilters().Parse(v)
			if err != nil {
				return err
			}

			qmod, _, _, err := filterer.ParseFilter(cond, {{.Column}}, op, rawValue, false)
			if err != nil {
				return err
			}
			qmods = append(qmods, qmod)
		}
	}
	{{else}}
	// TODO: Add support for {{.Type}} type for field {{.Name}}
	{{end}}
	{{end}}

	*q = append(*q, qmods...)

	return nil
}
{{if ne .SortField ""}}
// AddSorting adds the result of ParseSorting to a given query
func ({{.ReceiverName}} *{{.Name}}) AddSorting(query {{if eq $lib "bob"}}*[]bob.Mod[*dialect.SelectQuery]{{else if eq $lib "boiler"}}*[]qm.QueryMod{{end}}) error {
	{{if eq $lib "bob"}}filterer := bobops.BobFilterer{}{{else if eq $lib "boiler"}}filterer := boilerops.BoilFilterer{}{{end}}
	mod, err := filterer.ParseSorting({{$receiver}}.{{.SortField}})
	if err != nil {
		// If no sort parameters are passed, simply return the query as-is
		if errors.Is(err, ops.ErrEmptySort) {
			return nil
		}
		return err
	}
	*query = append(*query, mod)
	return nil
}
{{end}}
{{end}}
`
