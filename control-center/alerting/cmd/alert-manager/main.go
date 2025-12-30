package main

import (
	"flag"
	"log"
	"os"

	"github.com/paw/control-center/alerting"
	"github.com/paw/control-center/alerting/app"
	"github.com/paw/control-center/alerting/engine"
)

func main() {
	// Parse command-line flags
	configFile := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	// Set config file from flag if provided
	if *configFile != "" {
		os.Setenv("ALERTING_CONFIG_FILE", *configFile)
	}

	// Load configuration
	config, err := alerting.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create metrics provider (Prometheus)
	metricsProvider, err := engine.NewPrometheusProvider(config.PrometheusURL)
	if err != nil {
		log.Fatalf("Failed to create metrics provider: %v", err)
	}

	// Create and start server
	server, err := app.NewServer(config, metricsProvider)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	log.Println("PAW Alert Manager starting...")
	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
