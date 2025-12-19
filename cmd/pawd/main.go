package main

import (
	"os"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	"github.com/paw-chain/paw/app"
	"github.com/paw-chain/paw/cmd/pawd/cmd"
)

func main() {
	// Start Prometheus metrics server on port 36660
	// This runs in background goroutine
	StartPrometheusServer(36660)

	// Start health check server on port 36661
	// This provides /health, /health/ready, /health/detailed endpoints
	nodeChecker := NewSimpleNodeChecker("http://localhost:26657")
	StartHealthCheckServer(36661, nodeChecker)

	rootCmd := cmd.NewRootCmd(false)

	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		os.Exit(1)
	}
}
