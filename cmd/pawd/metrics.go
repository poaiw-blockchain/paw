package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// StartPrometheusServer starts a Prometheus metrics HTTP server on the given port.
// It runs in a background goroutine and logs startup failures.
// This is in addition to the SDK's built-in telemetry; useful for custom metrics.
func StartPrometheusServer(port int) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		// ListenAndServe blocks, so run in goroutine
		// Errors after startup (like port in use) are logged but not fatal
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("prometheus server error: %v\n", err)
		}
	}()
}
