package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"
)

// FunctionInfo holds information about a Go function
type FunctionInfo struct {
	Name       string
	Receiver   string
	Parameters []ParamInfo
	Returns    []string
	IsMethod   bool
	DocComment string
}

// ParamInfo holds parameter information
type ParamInfo struct {
	Name string
	Type string
}

// TestCaseTemplate holds test case information
type TestCaseTemplate struct {
	FuncName   string
	TestName   string
	TableName  string
	CaseCount  int
	Receiver   string
	ParamNames []string
	ParamTypes []string
	IsAsync    bool
	IsMethod   bool
	DocComment string
	EdgeCases  []string
}

// AnalyzeGoFile parses a Go file and extracts function information
func AnalyzeGoFile(filePath string) ([]FunctionInfo, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var functions []FunctionInfo

	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			funcInfo := extractFunctionInfo(funcDecl)
			functions = append(functions, funcInfo)
		}
		return true
	})

	return functions, nil
}

// extractFunctionInfo extracts information from a function declaration
func extractFunctionInfo(funcDecl *ast.FuncDecl) FunctionInfo {
	info := FunctionInfo{
		Name: funcDecl.Name.Name,
	}

	// Check if it's a method
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		info.IsMethod = true
		receiver := funcDecl.Recv.List[0]
		info.Receiver = getTypeString(receiver.Type)
	}

	// Extract parameters
	if funcDecl.Type.Params != nil {
		for _, param := range funcDecl.Type.Params.List {
			paramType := getTypeString(param.Type)
			if len(param.Names) == 0 {
				info.Parameters = append(info.Parameters, ParamInfo{Name: "", Type: paramType})
			} else {
				for _, name := range param.Names {
					info.Parameters = append(info.Parameters, ParamInfo{Name: name.Name, Type: paramType})
				}
			}
		}
	}

	// Extract return types
	if funcDecl.Type.Results != nil {
		for _, result := range funcDecl.Type.Results.List {
			returnType := getTypeString(result.Type)
			info.Returns = append(info.Returns, returnType)
		}
	}

	// Extract documentation
	if funcDecl.Doc != nil {
		info.DocComment = funcDecl.Doc.Text()
	}

	return info
}

// getTypeString converts an expression to a string representation
func getTypeString(expr ast.Expr) string {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.StarExpr:
		return "*" + getTypeString(x.X)
	case *ast.ArrayType:
		return "[]" + getTypeString(x.Elt)
	case *ast.MapType:
		return "map[" + getTypeString(x.Key) + "]" + getTypeString(x.Value)
	case *ast.SelectorExpr:
		return getTypeString(x.X) + "." + x.Sel.Name
	case *ast.InterfaceType:
		return "interface{}"
	default:
		return "interface{}"
	}
}

// GenerateTableDrivenTests generates table-driven test code
func GenerateTableDrivenTests(funcInfo *FunctionInfo) string {
	if funcInfo == nil {
		return ""
	}
	testName := "Test" + strings.ToUpper(string(funcInfo.Name[0])) + funcInfo.Name[1:]
	tableName := "cases"

	tmpl := `
// {{ .TestName }} tests {{ .FuncName }} with table-driven tests.
// This is a template - implement test cases to achieve 98%+ coverage.
func {{ .TestName }}(t *testing.T) {
	type args struct {
{{ range .ParamNames }}		{{ . }} interface{}
{{ end }}	}

	type {{ .TableName }} struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}

	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "basic case",
			args: args{
{{ range .ParamNames }}				{{ . }}: nil, // TODO: Add test value
{{ end }}			},
			want:    nil, // TODO: Add expected value
			wantErr: false,
		},
		{
			name: "empty input",
			args: args{
{{ range .ParamNames }}				{{ . }}: nil, // TODO: Add empty value
{{ end }}			},
			want:    nil, // TODO: Add expected value
			wantErr: false,
		},
		{
			name: "invalid input",
			args: args{
{{ range .ParamNames }}				{{ . }}: nil, // TODO: Add invalid value
{{ end }}			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "boundary condition",
			args: args{
{{ range .ParamNames }}				{{ . }}: nil, // TODO: Add boundary value
{{ end }}			},
			want:    nil, // TODO: Add expected value
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: Call the function and verify results
			// got, err := {{ .FuncName }}(tt.args...)
			// if (err != nil) != tt.wantErr {
			//     t.Errorf("{{ .FuncName }} error = %v, wantErr %v", err, tt.wantErr)
			// }
			// if got != tt.want {
			//     t.Errorf("{{ .FuncName }} = %v, want %v", got, tt.want)
			// }
		})
	}
}
`

	t, err := template.New("test").Parse(tmpl)
	if err != nil {
		return fmt.Sprintf("Error parsing template: %v", err)
	}

	testCase := TestCaseTemplate{
		TestName:   testName,
		FuncName:   funcInfo.Name,
		TableName:  tableName,
		ParamNames: getParamNames(funcInfo.Parameters),
	}

	var result strings.Builder
	err = t.Execute(&result, testCase)
	if err != nil {
		return fmt.Sprintf("Error executing template: %v", err)
	}

	return result.String()
}

// GenerateBenchmarkTests generates benchmark tests
func GenerateBenchmarkTests(funcInfo *FunctionInfo) string {
	if funcInfo == nil {
		return ""
	}
	benchName := "Benchmark" + strings.ToUpper(string(funcInfo.Name[0])) + funcInfo.Name[1:]

	return fmt.Sprintf(`
// %s benchmarks the performance of %s.
func %s(b *testing.B) {
	// TODO: Set up benchmark data

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// TODO: Call the function
		// _ = %s(...)
	}
}
`, benchName, funcInfo.Name, benchName, funcInfo.Name)
}

// GenerateTestFixtures generates test fixture functions
func GenerateTestFixtures(funcInfo *FunctionInfo) string {
	if funcInfo == nil {
		return ""
	}
	output := fmt.Sprintf("\n// Test fixtures for %s\n", funcInfo.Name)

	// Generate fixture for setup
	output += fmt.Sprintf(`
// setup%s sets up test data for %s tests.
func setup%s() interface{} {
	// TODO: Initialize test data
	return nil
}
`, strings.ToUpper(string(funcInfo.Name[0]))+funcInfo.Name[1:],
		funcInfo.Name,
		strings.ToUpper(string(funcInfo.Name[0]))+funcInfo.Name[1:])

	// Generate fixture for cleanup
	output += fmt.Sprintf(`
// cleanup%s cleans up test data.
func cleanup%s(data interface{}) {
	// TODO: Clean up test resources
}
`, strings.ToUpper(string(funcInfo.Name[0]))+funcInfo.Name[1:],
		strings.ToUpper(string(funcInfo.Name[0]))+funcInfo.Name[1:])

	return output
}

// GenerateEdgeCasePlaceholders generates placeholders for edge cases
func GenerateEdgeCasePlaceholders(funcInfo *FunctionInfo) string {
	if funcInfo == nil {
		return ""
	}
	output := "\n// Edge case tests\n"

	edgeCases := []string{
		"nil inputs",
		"empty values",
		"maximum values",
		"minimum values",
		"boundary conditions",
		"concurrent access",
		"invalid types",
		"missing fields",
	}

	for _, edgeCase := range edgeCases {
		testName := fmt.Sprintf("Test%s%s",
			strings.ToUpper(string(funcInfo.Name[0]))+funcInfo.Name[1:],
			strings.ReplaceAll(strings.ToUpper(edgeCase), " ", ""))

		output += fmt.Sprintf(`
// %s tests %s with %s.
// TODO: Implement test
func %s(t *testing.T) {
	t.Skip("TODO: Implement edge case test for %s")
}
`, testName, funcInfo.Name, edgeCase, testName, edgeCase)
	}

	return output
}

// getParamNames extracts parameter names from ParamInfo slice
func getParamNames(params []ParamInfo) []string {
	var names []string
	for _, p := range params {
		if p.Name != "" {
			names = append(names, p.Name)
		}
	}
	return names
}

// GenerateCompleteTestFile generates a complete test file
func GenerateCompleteTestFile(sourceFile string, functions []FunctionInfo) string {
	baseName := filepath.Base(sourceFile)
	packageName := strings.TrimSuffix(baseName, ".go")

	output := fmt.Sprintf(`package %s

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test file generated for %s
// This file provides test scaffolds for all public functions.
// Implement the TODO sections to achieve 98%% coverage.

`, packageName, sourceFile)

	// Add tests for each function
	for i := range functions {
		funcInfo := &functions[i]
		if funcInfo.Name != "" && unicode.IsUpper(rune(funcInfo.Name[0])) { // Only public functions
			output += GenerateTableDrivenTests(funcInfo)
			output += GenerateBenchmarkTests(funcInfo)
			output += GenerateTestFixtures(funcInfo)
			output += GenerateEdgeCasePlaceholders(funcInfo)
		}
	}

	// Add general test helpers
	output += `

// TestHelpers provides common testing utilities

// AssertEqual is a helper function for assertions
func AssertEqual(t *testing.T, expected, actual interface{}, message string) {
	if expected != actual {
		t.Errorf("Assertion failed: %s (expected %v, got %v)", message, expected, actual)
	}
}

// AssertNoError is a helper function for error assertions
func AssertNoError(t *testing.T, err error, message string) {
	if err != nil {
		t.Errorf("Assertion failed: %s (error: %v)", message, err)
	}
}

// AssertError is a helper function to assert an error occurred
func AssertError(t *testing.T, err error, message string) {
	if err == nil {
		t.Errorf("Assertion failed: %s (expected error but got none)", message)
	}
}
`

	return output
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: go_test_generator <go-file> [--output <test-file>]\n")
		fmt.Fprintf(os.Stderr, "Example: go_test_generator main.go\n")
		os.Exit(1)
	}

	sourceFile := os.Args[1]
	outputFile := strings.TrimSuffix(sourceFile, ".go") + "_test.go"

	if len(os.Args) > 3 && os.Args[2] == "--output" {
		outputFile = os.Args[3]
	}

	// Validate input file
	if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
		log.Fatalf("Source file not found: %s", sourceFile)
	}

	// Analyze Go file
	fmt.Printf("Analyzing %s...\n", sourceFile)
	functions, err := AnalyzeGoFile(sourceFile)
	if err != nil {
		log.Fatalf("Error analyzing file: %v", err)
	}

	if len(functions) == 0 {
		fmt.Println("No functions found in file")
		return
	}

	// Generate test file
	fmt.Printf("Found %d functions\n", len(functions))
	testContent := GenerateCompleteTestFile(sourceFile, functions)

	// Write output file
	if err := os.WriteFile(outputFile, []byte(testContent), 0o600); err != nil {
		log.Fatalf("Error writing test file: %v", err)
	}

	fmt.Printf("Test file generated: %s\n", outputFile)
	fmt.Printf("Lines of code: %d\n", strings.Count(testContent, "\n"))
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("1. Edit %s to implement test cases\n", outputFile)
	fmt.Printf("2. Run: go test -v\n")
	fmt.Printf("3. Run: go test -cover\n")
}
