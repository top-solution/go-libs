package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/top-solution/go-libs/v2/dbutils/ops/gen"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Usage: gen <filter_type> <root_path>")
	}

	filterType := os.Args[1]
	rootPath := os.Args[2]

	// Convert relative path to absolute for better handling
	absRootPath, err := filepath.Abs(rootPath)
	if err != nil {
		log.Fatalf("Failed to get absolute path for %s: %v", rootPath, err)
	}

	fmt.Printf("Scanning directory: %s\n", absRootPath)

	// Walk through all directories under the root path
	err = filepath.Walk(absRootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a directory
		if !info.IsDir() {
			return nil
		}

		// Skip hidden directories and vendor directories, but allow the root directory even if it starts with "."
		if path != absRootPath && (strings.HasPrefix(info.Name(), ".") || info.Name() == "vendor") {
			return filepath.SkipDir
		}

		// Check if this directory contains Go files (excluding test and generated files)
		hasGoFiles, err := hasRelevantGoFiles(path)
		if err != nil {
			return err
		}

		if !hasGoFiles {
			return nil
		}

		// Get package name from directory name
		packageName := filepath.Base(path)

		// Handle special case where the directory is "." or the root
		if packageName == "." || path == absRootPath {
			// Try to get package name from go.mod or use directory name
			if wd, err := os.Getwd(); err == nil {
				packageName = filepath.Base(wd)
			}
		}

		fmt.Printf("Processing package: %s (path: %s)\n", packageName, path)

		// Create generator and process the package
		generator := gen.NewGenerator(packageName, path, filterType)
		if err := generator.GenerateFromPackage(); err != nil {
			log.Printf("Warning: Failed to generate filters for package %s: %v", path, err)
			return nil // Continue processing other packages
		}

		return nil
	})

	if err != nil {
		log.Fatalf("Failed to walk directory tree: %v", err)
	}

	fmt.Println("Filter generation completed.")
}

// hasRelevantGoFiles checks if a directory contains Go files that are not test files or generated files
func hasRelevantGoFiles(dir string) (bool, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return false, err
	}

	for _, file := range files {
		filename := filepath.Base(file)
		// Skip test files and generated files
		if !strings.HasSuffix(filename, "_test.go") && !strings.Contains(filename, "_gen.go") {
			return true, nil
		}
	}

	return false, nil
}
