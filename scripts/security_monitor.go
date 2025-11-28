package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

const (
	version = "1.0.0"
)

// Severity levels
const (
	SeverityCritical = "CRITICAL"
	SeverityHigh     = "HIGH"
	SeverityMedium   = "MEDIUM"
	SeverityLow      = "LOW"
	SeverityInfo     = "INFO"
)

// SecurityFinding represents a single security finding
type SecurityFinding struct {
	Tool        string  `json:"tool"`
	Severity    string  `json:"severity"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Location    string  `json:"location"`
	CWE         *string `json:"cwe,omitempty"`
	Remediation *string `json:"remediation,omitempty"`
	Timestamp   string  `json:"timestamp"`
}

// ScanReport represents the overall scan report
type ScanReport struct {
	ScanTimestamp string            `json:"scan_timestamp"`
	TotalFindings int               `json:"total_findings"`
	BySeverity    map[string]int    `json:"by_severity"`
	ByTool        map[string]int    `json:"by_tool"`
	Findings      []SecurityFinding `json:"findings"`
}

// SecurityScanner orchestrates all security scans
type SecurityScanner struct {
	findings  []SecurityFinding
	timestamp string
	workDir   string
}

// NewSecurityScanner creates a new scanner instance
func NewSecurityScanner() *SecurityScanner {
	return &SecurityScanner{
		findings:  []SecurityFinding{},
		timestamp: time.Now().UTC().Format(time.RFC3339),
		workDir:   ".",
	}
}

// RunAllScans executes all enabled security scanners
func (s *SecurityScanner) RunAllScans() error {
	log.Println("Starting comprehensive Go security scanning...")

	allPassed := true

	// Run GoSec
	if err := s.runGoSec(); err != nil {
		log.Printf("GoSec failed: %v", err)
		allPassed = false
	}

	// Run Nancy
	if err := s.runNancy(); err != nil {
		log.Printf("Nancy failed: %v", err)
		allPassed = false
	}

	// Run govulncheck
	if err := s.runGovulncheck(); err != nil {
		log.Printf("govulncheck failed: %v", err)
		allPassed = false
	}

	// Run golangci-lint with security checks
	if err := s.runGolangciLint(); err != nil {
		log.Printf("golangci-lint failed: %v", err)
		allPassed = false
	}

	// Generate reports
	if err := s.generateReports(); err != nil {
		log.Printf("Report generation failed: %v", err)
		allPassed = false
	}

	if !allPassed {
		return fmt.Errorf("one or more security scans failed")
	}

	return nil
}

// runGoSec executes the GoSec security scanner
func (s *SecurityScanner) runGoSec() error {
	log.Println("Running GoSec security scanner...")

	cmd := exec.Command("gosec",
		"-conf", ".security/.gosec.yml",
		"-fmt", "json",
		"-out", "gosec-report.json",
		"./...")
	cmd.Dir = s.workDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// GoSec returns error if findings exist
		if stderr.Len() > 0 {
			log.Printf("GoSec stderr: %s", stderr.String())
		}
	}

	// Parse results
	data, err := os.ReadFile("gosec-report.json")
	if err != nil {
		log.Printf("Could not read gosec-report.json: %v", err)
		return nil // Don't fail if file doesn't exist
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("failed to parse gosec report: %w", err)
	}

	issues, ok := result["Issues"].([]interface{})
	if ok {
		for _, issue := range issues {
			if issueMap, ok := issue.(map[string]interface{}); ok {
				severity := "MEDIUM"
				if sev, ok := issueMap["severity"].(string); ok {
					severity = sev
				}

				location := ""
				if file, ok := issueMap["file"].(string); ok {
					location = file
					if line, ok := issueMap["line"].(float64); ok {
						location = fmt.Sprintf("%s:%d", file, int(line))
					}
				}

				finding := SecurityFinding{
					Tool:        "gosec",
					Severity:    severity,
					Title:       "Security Issue",
					Description: getString(issueMap, "details"),
					Location:    location,
					CWE:         getStringPtr(issueMap, "cwe"),
					Timestamp:   s.timestamp,
				}
				s.findings = append(s.findings, finding)
			}
		}
	}

	log.Println("GoSec scan completed")
	return nil
}

// runNancy executes the Nancy dependency checker
func (s *SecurityScanner) runNancy() error {
	log.Println("Running Nancy dependency scanner...")

	cmd := exec.Command("sh", "-c", "go list -json -m all | nancy sleuth -o json > nancy-report.json")
	cmd.Dir = s.workDir

	if err := cmd.Run(); err != nil {
		log.Printf("Nancy execution failed: %v", err)
		// Continue even if nancy fails
	}

	data, err := os.ReadFile("nancy-report.json")
	if err != nil {
		log.Printf("Could not read nancy-report.json: %v", err)
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("failed to parse nancy report: %w", err)
	}

	// Parse Nancy results (format varies)
	if vulns, ok := result["vulnerabilities"].([]interface{}); ok {
		for _, vuln := range vulns {
			if vulnMap, ok := vuln.(map[string]interface{}); ok {
				finding := SecurityFinding{
					Tool:        "nancy",
					Severity:    "HIGH",
					Title:       getString(vulnMap, "package"),
					Description: getString(vulnMap, "vulnerability"),
					Location:    "dependencies",
					Timestamp:   s.timestamp,
				}
				s.findings = append(s.findings, finding)
			}
		}
	}

	log.Println("Nancy scan completed")
	return nil
}

// runGovulncheck executes the Go vulnerability database checker
func (s *SecurityScanner) runGovulncheck() error {
	log.Println("Running govulncheck...")

	cmd := exec.Command("govulncheck", "-json", "./...")
	cmd.Dir = s.workDir

	output, err := cmd.Output()
	if err != nil {
		// govulncheck returns error if vulnerabilities found
		log.Printf("govulncheck found issues")
	}

	// Parse line-delimited JSON output
	for _, line := range strings.Split(string(output), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var result map[string]interface{}
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			continue
		}

		if vulnType, ok := result["type"].(string); ok && vulnType == "Vuln" {
			finding := SecurityFinding{
				Tool:        "govulncheck",
				Severity:    "HIGH",
				Title:       getString(result, "vuln_id"),
				Description: getString(result, "summary"),
				Location:    getString(result, "package"),
				Timestamp:   s.timestamp,
			}
			s.findings = append(s.findings, finding)
		}
	}

	log.Println("govulncheck completed")
	return nil
}

// runGolangciLint executes golangci-lint with security checks
func (s *SecurityScanner) runGolangciLint() error {
	log.Println("Running golangci-lint with security checks...")

	cmd := exec.Command("golangci-lint",
		"run",
		"--config", ".golangci.yml",
		"--out-format", "json",
		"-o", "golangci-report.json",
		"./...")
	cmd.Dir = s.workDir

	if err := cmd.Run(); err != nil {
		// golangci-lint returns error if issues found
		log.Printf("golangci-lint found issues")
	}

	data, err := os.ReadFile("golangci-report.json")
	if err != nil {
		log.Printf("Could not read golangci-report.json: %v", err)
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("failed to parse golangci report: %w", err)
	}

	issues, ok := result["Issues"].([]interface{})
	if ok {
		for _, issue := range issues {
			if issueMap, ok := issue.(map[string]interface{}); ok {
				// Only include security-related checks
				linter := getString(issueMap, "from_linter")
				if !isSecurityLinter(linter) {
					continue
				}

				location := fmt.Sprintf("%s:%v", getString(issueMap, "pos"), "")
				finding := SecurityFinding{
					Tool:        "golangci-lint",
					Severity:    "MEDIUM",
					Title:       linter,
					Description: getString(issueMap, "text"),
					Location:    location,
					Timestamp:   s.timestamp,
				}
				s.findings = append(s.findings, finding)
			}
		}
	}

	log.Println("golangci-lint scan completed")
	return nil
}

// isSecurityLinter checks if the linter is security-related
func isSecurityLinter(linter string) bool {
	securityLinters := map[string]bool{
		"gosec":       true,
		"staticcheck": true,
		"gas":         true,
	}
	return securityLinters[linter]
}

// generateReports generates various report formats
func (s *SecurityScanner) generateReports() error {
	log.Println("Generating security reports...")

	// Count by severity
	bySeverity := make(map[string]int)
	for _, finding := range s.findings {
		bySeverity[finding.Severity]++
	}

	// Count by tool
	byTool := make(map[string]int)
	for _, finding := range s.findings {
		byTool[finding.Tool]++
	}

	report := ScanReport{
		ScanTimestamp: s.timestamp,
		TotalFindings: len(s.findings),
		BySeverity:    bySeverity,
		ByTool:        byTool,
		Findings:      s.findings,
	}

	// Generate JSON report
	if err := s.generateJSONReport(report); err != nil {
		return fmt.Errorf("failed to generate JSON report: %w", err)
	}

	// Generate Markdown report
	if err := s.generateMarkdownReport(report); err != nil {
		return fmt.Errorf("failed to generate Markdown report: %w", err)
	}

	// Generate SARIF report for 
	if err := s.generateSARIFReport(report); err != nil {
		return fmt.Errorf("failed to generate SARIF report: %w", err)
	}

	return nil
}

// generateJSONReport exports findings as JSON
func (s *SecurityScanner) generateJSONReport(report ScanReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile("security-report.json", data, 0644)
}

// generateMarkdownReport generates a human-readable markdown report
func (s *SecurityScanner) generateMarkdownReport(report ScanReport) error {
	var buf bytes.Buffer

	buf.WriteString("# Go Security Scan Report\n\n")
	buf.WriteString(fmt.Sprintf("**Scan Timestamp:** %s\n\n", report.ScanTimestamp))
	buf.WriteString(fmt.Sprintf("**Total Findings:** %d\n\n", report.TotalFindings))

	// Summary by severity
	buf.WriteString("## Summary by Severity\n\n")
	for _, severity := range []string{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo} {
		if count, ok := report.BySeverity[severity]; ok && count > 0 {
			buf.WriteString(fmt.Sprintf("- **%s:** %d\n", severity, count))
		}
	}

	// Summary by tool
	buf.WriteString("\n## Summary by Tool\n\n")
	tools := make([]string, 0, len(report.ByTool))
	for tool := range report.ByTool {
		tools = append(tools, tool)
	}
	sort.Strings(tools)

	for _, tool := range tools {
		buf.WriteString(fmt.Sprintf("- **%s:** %d\n", tool, report.ByTool[tool]))
	}

	// Detailed findings
	if len(report.Findings) > 0 {
		buf.WriteString("\n## Detailed Findings\n\n")
		for _, finding := range report.Findings {
			buf.WriteString(fmt.Sprintf("### %s\n\n", finding.Title))
			buf.WriteString(fmt.Sprintf("- **Tool:** %s\n", finding.Tool))
			buf.WriteString(fmt.Sprintf("- **Severity:** %s\n", finding.Severity))
			buf.WriteString(fmt.Sprintf("- **Location:** %s\n", finding.Location))
			if finding.Description != "" {
				buf.WriteString(fmt.Sprintf("- **Description:** %s\n", finding.Description))
			}
			if finding.CWE != nil && *finding.CWE != "" {
				buf.WriteString(fmt.Sprintf("- **CWE:** %s\n", *finding.CWE))
			}
			if finding.Remediation != nil && *finding.Remediation != "" {
				buf.WriteString(fmt.Sprintf("- **Remediation:** %s\n", *finding.Remediation))
			}
			buf.WriteString("\n")
		}
	}

	return os.WriteFile("security-report.md", buf.Bytes(), 0644)
}

// generateSARIFReport generates SARIF format for  integration
func (s *SecurityScanner) generateSARIFReport(report ScanReport) error {
	sarif := map[string]interface{}{
		"version": "2.1.0",
		"runs": []map[string]interface{}{
			{
				"tool": map[string]interface{}{
					"driver": map[string]interface{}{
						"name":           "PAW Security Scanner",
						"version":        version,
						"informationUri": "https://github.com/paw-org/paw",
					},
				},
				"results": s.buildSARIFResults(),
			},
		},
	}

	data, err := json.MarshalIndent(sarif, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile("security-report.sarif", data, 0644)
}

// buildSARIFResults converts findings to SARIF results format
func (s *SecurityScanner) buildSARIFResults() []map[string]interface{} {
	var results []map[string]interface{}

	for _, finding := range s.findings {
		parts := strings.Split(finding.Location, ":")
		file := parts[0]
		line := 1
		if len(parts) > 1 {
			fmt.Sscanf(parts[1], "%d", &line)
		}

		result := map[string]interface{}{
			"ruleId": finding.Tool,
			"level":  strings.ToLower(finding.Severity),
			"message": map[string]interface{}{
				"text": finding.Title,
			},
			"locations": []map[string]interface{}{
				{
					"physicalLocation": map[string]interface{}{
						"artifactLocation": map[string]string{
							"uri": file,
						},
						"region": map[string]int{
							"startLine": line,
						},
					},
				},
			},
		}

		results = append(results, result)
	}

	return results
}

// Helper functions

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getStringPtr(m map[string]interface{}, key string) *string {
	val := getString(m, key)
	if val != "" {
		return &val
	}
	return nil
}

func main() {
	flag.Parse()

	scanner := NewSecurityScanner()

	if err := scanner.RunAllScans(); err != nil {
		log.Printf("Security scanning failed: %v", err)
		os.Exit(1)
	}

	log.Println("Security scanning completed successfully")
	os.Exit(0)
}
