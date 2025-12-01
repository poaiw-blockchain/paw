package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"path/filepath"
	"strings"
)

// Directories to process
var targetDirs = []string{
	"tests/property",
	"tests/differential",
	"tests/security",
	"tests/integration",
	"tests/ibc",
}

// Excluded test patterns (tests that must run sequentially)
var excludedPatterns = []string{
	"Suite",            // Suite tests share state
	"Benchmark",        // Benchmarks don't need t.Parallel()
	"Example",          // Examples don't need t.Parallel()
	"NetworkPartition", // Network tests may share state
	"TestIBC",          // IBC tests use coordinator (shared state)
}

func main() {
	rootDir := "/home/decri/blockchain-projects/paw"

	for _, dir := range targetDirs {
		dirPath := filepath.Join(rootDir, dir)
		err := processDirectory(dirPath)
		if err != nil {
			fmt.Printf("Error processing %s: %v\n", dirPath, err)
		}
	}

	fmt.Println("Successfully added t.Parallel() to eligible test functions!")
}

func processDirectory(dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		return err
	}

	for _, file := range files {
		if err := processFile(file); err != nil {
			fmt.Printf("Error processing %s: %v\n", file, err)
		}
	}

	return nil
}

func processFile(filename string) error {
	fset := token.NewFileSet()

	// Read file content
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	// Parse the file
	f, err := parser.ParseFile(fset, filename, content, parser.ParseComments)
	if err != nil {
		return err
	}

	modified := false

	// Find all test functions
	ast.Inspect(f, func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		// Check if it's a test function
		if !isTestFunction(funcDecl) {
			return true
		}

		// Check if excluded
		if shouldExclude(funcDecl.Name.Name) {
			return true
		}

		// Check if t.Parallel() already exists
		if hasParallelCall(funcDecl) {
			return true
		}

		// Add t.Parallel() as first statement
		addParallelCall(funcDecl)
		modified = true

		fmt.Printf("Added t.Parallel() to %s in %s\n", funcDecl.Name.Name, filepath.Base(filename))

		return true
	})

	// Write back if modified
	if modified {
		var buf bytes.Buffer
		if err := printer.Fprint(&buf, fset, f); err != nil {
			return err
		}

		if err := ioutil.WriteFile(filename, buf.Bytes(), 0644); err != nil {
			return err
		}
	}

	return nil
}

func isTestFunction(funcDecl *ast.FuncDecl) bool {
	name := funcDecl.Name.Name

	// Must start with Test
	if !strings.HasPrefix(name, "Test") {
		return false
	}

	// Must have one parameter of type *testing.T
	if funcDecl.Type.Params == nil || len(funcDecl.Type.Params.List) != 1 {
		return false
	}

	param := funcDecl.Type.Params.List[0]
	starExpr, ok := param.Type.(*ast.StarExpr)
	if !ok {
		return false
	}

	selExpr, ok := starExpr.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := selExpr.X.(*ast.Ident)
	if !ok || ident.Name != "testing" {
		return false
	}

	return selExpr.Sel.Name == "T"
}

func shouldExclude(funcName string) bool {
	for _, pattern := range excludedPatterns {
		if strings.Contains(funcName, pattern) {
			return true
		}
	}
	return false
}

func hasParallelCall(funcDecl *ast.FuncDecl) bool {
	if funcDecl.Body == nil {
		return false
	}

	for _, stmt := range funcDecl.Body.List {
		exprStmt, ok := stmt.(*ast.ExprStmt)
		if !ok {
			continue
		}

		callExpr, ok := exprStmt.X.(*ast.CallExpr)
		if !ok {
			continue
		}

		selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			continue
		}

		if ident, ok := selExpr.X.(*ast.Ident); ok {
			if ident.Name == "t" && selExpr.Sel.Name == "Parallel" {
				return true
			}
		}
	}

	return false
}

func addParallelCall(funcDecl *ast.FuncDecl) {
	if funcDecl.Body == nil {
		return
	}

	// Create t.Parallel() call
	parallelCall := &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent("t"),
				Sel: ast.NewIdent("Parallel"),
			},
		},
	}

	// Insert as first statement
	funcDecl.Body.List = append([]ast.Stmt{parallelCall}, funcDecl.Body.List...)
}
