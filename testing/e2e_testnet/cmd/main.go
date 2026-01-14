// PAW Testnet E2E Validation Tool
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	e2e "e2e_testnet"
)

func main() {
	var (
		verbose     = flag.Bool("v", false, "Verbose output")
		outputDir   = flag.String("output", "", "Output directory for results (default: ~/testnets/paw-mvp-1/results)")
		jsonOutput  = flag.Bool("json", false, "Output results as JSON")
		mdOutput    = flag.Bool("md", true, "Output results as Markdown")
		timeout     = flag.Duration("timeout", 5*time.Minute, "Test timeout")
		allTests    = flag.Bool("all", false, "Run all validation tests (phases 1-5)")
		localOnly   = flag.Bool("local", true, "Test only local validators (no SSH to other servers)")
		fullNetwork = flag.Bool("full", false, "Test all validators including remote (requires SSH)")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "PAW Testnet E2E Validation Tool\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -all                     # Run all tests (local validators only)\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -all -full               # Run all tests on ALL validators (requires SSH)\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nDefault: Tests only local validators (val1, val2 on this server)\n")
		fmt.Fprintf(os.Stderr, "Results saved to: ~/testnets/paw-mvp-1/results/\n")
	}

	flag.Parse()

	var cfg *e2e.TestnetConfig
	if *fullNetwork {
		cfg = e2e.DefaultTestnetConfig()
	} else if *localOnly {
		cfg = e2e.LocalOnlyConfig()
	} else {
		cfg = e2e.LocalOnlyConfig()
	}
	e2e.LoadConfigFromEnv(cfg)

	if *timeout > 0 {
		cfg.Timeout = *timeout
	}

	if *outputDir == "" {
		*outputDir = e2e.DefaultOutputDir()
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	if *allTests {
		runner := e2e.NewRunner(cfg, *verbose)
		suite := runner.RunAll(ctx)

		timestamp := time.Now().Format("20060102-150405")

		if *jsonOutput {
			jsonPath := filepath.Join(*outputDir, fmt.Sprintf("validation-%s.json", timestamp))
			if err := runner.SaveResults(jsonPath); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to save JSON: %v\n", err)
			} else {
				fmt.Printf("Results saved to: %s\n", jsonPath)
			}
		}

		if *mdOutput {
			mdPath := filepath.Join(*outputDir, fmt.Sprintf("VALIDATION-%s.md", timestamp))
			if err := runner.SaveMarkdown(mdPath); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to save Markdown: %v\n", err)
			} else {
				fmt.Printf("Report saved to: %s\n", mdPath)
			}
		}

		if suite.FailedTests > 0 {
			os.Exit(1)
		}
		return
	}

	fmt.Println("Use -all to run validation tests")
	fmt.Println("Use -help for more options")
}
