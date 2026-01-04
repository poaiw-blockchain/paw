package main

import (
	"os"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	"github.com/paw-chain/paw/app"
	"github.com/paw-chain/paw/cmd/pawd/cmd"
)

func main() {
	home := resolveNodeHome(os.Args[1:])
	metricsPort, healthPort := loadTelemetryPorts(home)
	rpcEndpoint := resolveRPCAddress(home)

	// Start Prometheus metrics server on the configured port.
	StartPrometheusServer(metricsPort)

	// Start health check server with the configured port + RPC endpoint.
	nodeChecker := NewSimpleNodeChecker(rpcEndpoint)
	StartHealthCheckServer(healthPort, nodeChecker)

	rootCmd := cmd.NewRootCmd(false)

	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		os.Exit(1)
	}
}
