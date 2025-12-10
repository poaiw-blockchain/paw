package main

import (
	"fmt"
	"os"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	"github.com/paw-chain/paw/app"
	"github.com/paw-chain/paw/cmd/pawd/cmd"
)

func main() {
	// Start Prometheus metrics server on port 36660
	// This runs in background goroutine
	if err := StartPrometheusServer(36660); err != nil {
		// Log error but don't fail - metrics are optional
		if _, writeErr := os.Stderr.WriteString("Warning: Failed to start Prometheus metrics server\n"); writeErr != nil {
			fmt.Fprintf(os.Stderr, "failed to log metrics warning: %v\n", writeErr)
		}
	}

	rootCmd := cmd.NewRootCmd()

	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		os.Exit(1)
	}
}
