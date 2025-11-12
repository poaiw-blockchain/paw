// Crypto Check - Custom cryptographic usage analyzer for PAW Blockchain
// This tool scans the codebase for cryptographic issues and best practices
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// Issue represents a potential security issue found
type Issue struct {
	File     string
	Line     int
	Column   int
	Severity string
	Category string
	Message  string
}

var (
	issues []Issue
	fset   = token.NewFileSet()
)

// Weak crypto packages to detect
var weakCryptoPackages = map[string]string{
	"crypto/md5":  "MD5 is cryptographically broken and should not be used",
	"crypto/sha1": "SHA1 is cryptographically weak and should be avoided",
	"crypto/des":  "DES is obsolete and should not be used",
	"crypto/rc4":  "RC4 is broken and should not be used",
}

// Insecure random packages
var insecureRandom = map[string]string{
	"math/rand": "math/rand is not cryptographically secure, use crypto/rand instead",
}

// Required crypto packages for blockchain
var requiredCryptoPackages = []string{
	"crypto/rand",
	"crypto/sha256",
	"crypto/sha512",
	"crypto/ecdsa",
	"crypto/ed25519",
}

func main() {
	if len(os.Args) < 2 {
		// Default to current directory
		if err := scanDirectory("."); err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning directory: %v\n", err)
			os.Exit(1)
		}
	} else {
		for _, arg := range os.Args[1:] {
			if err := scanDirectory(arg); err != nil {
				fmt.Fprintf(os.Stderr, "Error scanning %s: %v\n", arg, err)
				os.Exit(1)
			}
		}
	}

	printResults()

	if hasHighSeverityIssues() {
		os.Exit(1)
	}
}

func scanDirectory(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor, .git, and test files for security checks
		if info.IsDir() {
			if info.Name() == "vendor" || info.Name() == ".git" || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process Go files (skip test files for some checks)
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		return scanFile(path)
	})
}

func scanFile(filename string) error {
	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	// Check imports
	checkImports(file, filename)

	// Check for hardcoded secrets
	checkHardcodedSecrets(file, filename)

	// Check random number generation
	checkRandomUsage(file, filename)

	// Check crypto key sizes
	checkKeySizes(file, filename)

	// Check TLS configuration
	checkTLSConfig(file, filename)

	return nil
}

func checkImports(file *ast.File, filename string) {
	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)

		// Check for weak crypto
		if msg, found := weakCryptoPackages[importPath]; found {
			addIssue(filename, fset.Position(imp.Pos()).Line, fset.Position(imp.Pos()).Column,
				"HIGH", "Weak Cryptography", msg)
		}

		// Check for insecure random
		if msg, found := insecureRandom[importPath]; found {
			// Only warn if not a test file
			if !strings.HasSuffix(filename, "_test.go") {
				addIssue(filename, fset.Position(imp.Pos()).Line, fset.Position(imp.Pos()).Column,
					"MEDIUM", "Insecure Random", msg)
			}
		}
	}
}

func checkHardcodedSecrets(file *ast.File, filename string) {
	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BasicLit:
			if x.Kind == token.STRING {
				value := strings.ToLower(x.Value)

				// Check for potential hardcoded secrets
				secretKeywords := []string{"password", "secret", "api_key", "apikey", "private_key", "privatekey", "token"}
				for _, keyword := range secretKeywords {
					if strings.Contains(value, keyword) && len(x.Value) > len(keyword)+10 {
						addIssue(filename, fset.Position(x.Pos()).Line, fset.Position(x.Pos()).Column,
							"HIGH", "Hardcoded Secret", fmt.Sprintf("Potential hardcoded secret containing '%s'", keyword))
					}
				}

				// Check for hex-encoded strings that might be keys (common pattern)
				if len(x.Value) > 32 && isHexString(strings.Trim(x.Value, `"`)) {
					addIssue(filename, fset.Position(x.Pos()).Line, fset.Position(x.Pos()).Column,
						"MEDIUM", "Potential Key", "Long hex string that might be a hardcoded key")
				}
			}
		}
		return true
	})
}

func checkRandomUsage(file *ast.File, filename string) {
	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok {
					// Check for rand.Read or rand.Int usage
					if ident.Name == "rand" && (sel.Sel.Name == "Read" || sel.Sel.Name == "Int") {
						// Need to determine if this is math/rand or crypto/rand
						// This is a simplified check
						if !strings.HasSuffix(filename, "_test.go") {
							addIssue(filename, fset.Position(x.Pos()).Line, fset.Position(x.Pos()).Column,
								"MEDIUM", "Random Number Generation", "Verify this uses crypto/rand, not math/rand")
						}
					}
				}
			}
		}
		return true
	})
}

func checkKeySizes(file *ast.File, filename string) {
	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BasicLit:
			if x.Kind == token.INT {
				// Check for small key sizes
				if x.Value == "1024" || x.Value == "512" {
					addIssue(filename, fset.Position(x.Pos()).Line, fset.Position(x.Pos()).Column,
						"MEDIUM", "Key Size", "Potential weak key size (should be at least 2048 for RSA)")
				}
			}
		}
		return true
	})
}

func checkTLSConfig(file *ast.File, filename string) {
	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.KeyValueExpr:
			if ident, ok := x.Key.(*ast.Ident); ok {
				if ident.Name == "InsecureSkipVerify" {
					if basicLit, ok := x.Value.(*ast.Ident); ok {
						if basicLit.Name == "true" {
							addIssue(filename, fset.Position(x.Pos()).Line, fset.Position(x.Pos()).Column,
								"HIGH", "TLS Configuration", "InsecureSkipVerify is set to true - this is insecure")
						}
					}
				}
			}
		}
		return true
	})
}

func isHexString(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func addIssue(file string, line, column int, severity, category, message string) {
	issues = append(issues, Issue{
		File:     file,
		Line:     line,
		Column:   column,
		Severity: severity,
		Category: category,
		Message:  message,
	})
}

func printResults() {
	if len(issues) == 0 {
		fmt.Println("✓ No cryptographic issues found")
		return
	}

	fmt.Printf("\nFound %d potential cryptographic issues:\n\n", len(issues))

	// Group by severity
	high := []Issue{}
	medium := []Issue{}
	low := []Issue{}

	for _, issue := range issues {
		switch issue.Severity {
		case "HIGH":
			high = append(high, issue)
		case "MEDIUM":
			medium = append(medium, issue)
		case "LOW":
			low = append(low, issue)
		}
	}

	printIssueGroup("HIGH SEVERITY", high)
	printIssueGroup("MEDIUM SEVERITY", medium)
	printIssueGroup("LOW SEVERITY", low)
}

func printIssueGroup(title string, issues []Issue) {
	if len(issues) == 0 {
		return
	}

	fmt.Printf("=== %s ===\n", title)
	for _, issue := range issues {
		fmt.Printf("%s:%d:%d [%s] %s\n",
			issue.File, issue.Line, issue.Column, issue.Category, issue.Message)
	}
	fmt.Println()
}

func hasHighSeverityIssues() bool {
	for _, issue := range issues {
		if issue.Severity == "HIGH" {
			return true
		}
	}
	return false
}
