# Filter Code Generator

This package provides automatic code generation for database filter methods based on struct field comments.

## Overview

Instead of manually creating `FilterMap` instances and calling `AddFilters` for each API filter, you can now:

1. Define your endpoint payload struct with special `db:filter` comments
2. Run `go generate` to automatically create `AddFilters` methods
3. Use the generated methods in your handlers

## Usage

### 1. Annotate your structs

Add `db:filter` comments to fields that should be filterable:

```go
// db:filter
// db:filter import "fmt"   <----- this is optional, to add custom imports in the generated file. Can be repeated.
// db:filter sortField Sort <----- this is optional, to also generate a sorting func
type ListDCRsRequest struct {
    // db:filter bob_gen.ColumnNames.DCRS.Type
    Type   string `query:"type"`
    // db:filter bob_gen.ColumnNames.DCRS.Status sortBy "dcrs.status_order"
    Status string `query:"status"`  // <----- sortBy is optional, to specify a different column for sorting
    // db:filter bob_gen.ColumnNames.DCRS.CreatedBy
    CreatedBy *string `query:"created_by"`
    // db:filter bob_gen.ColumnNames.DCRS.Tags
    Tags []string `query:"tags"`

    // Regular fields without filter comments are ignored
    Limit  int `query:"limit"`
    Offset int `query:"offset"`

    Sort []string `query:"sort"`  // <----- this field is referenced by sortField
}
```

Of course, this is assuming Huma. There is no support for Goa, sorry.

**Note on `sortBy`**: When you specify `sortBy` on a field, the generator creates a separate `SortColumnsMap` that maps query parameters to their sort columns. This is useful when the column you want to sort by is different from the column you filter on. Fields without `sortBy` will use their filter column for sorting.

### 2. Add go generate directive

Add this line to the top of your model files (it will also work in main.go, only a bit slower):

```go
//go:generate go run github.com/top-solution/go-libs/v2/dbutils/ops/gen/cmd bob .
```

Or use the command directly:

```bash
# Scan all folders inside ., generate bob filters
go run github.com/top-solution/go-libs/v2/dbutils/ops/gen/cmd bob .

# Scan specific package, generate boiler filters
go run github.com/top-solution/go-libs/v2/dbutils/ops/gen/cmd boiler path/to/specific/packagh
```

### 3. Run go generate

```bash
go generate ./...
```

Using ./.. will make sure it's going to also run //go:generate directive inside your model files.

### 4. Use the generated methods

The generator creates an `AddFilters` method for each annotated struct:

```go
func (r *ListDCRsRequest) AddFilters(q *[]bob.Mod[*dialect.SelectQuery]) error
```

Use it in your handlers:

```go
func ListDCRsHandler(ctx context.Context, req *ListDCRsRequest) (*ListDCRsResponse, error) {
    var query []bob.Mod[*dialect.SelectQuery]
    
    // Automatically add filters based on request fields
    if err := req.AddFilters(&query); err != nil {
        return nil, err
    }
    
    // Add other query modifications
    query = append(query, sm.Limit(req.Limit))
    
    // Execute query
    dcrs, err := models.DCRS(query...).All(ctx, db)
    // ...
}
```
