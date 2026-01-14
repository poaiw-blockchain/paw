package main

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"

	"github.com/paw-chain/paw/app"
)

const (
	defaultMetricsPort = 36660
	defaultHealthPort  = 36661
	defaultRPCAddress  = "http://127.0.0.1:26657"
)

// resolveNodeHome returns the configured PAW home directory.
// It honors PAW_HOME and the --home flag if provided.
func resolveNodeHome(args []string) string {
	if home := os.Getenv("PAW_HOME"); home != "" {
		return home
	}

	for i, arg := range args {
		if strings.HasPrefix(arg, "--home=") {
			return strings.SplitN(arg, "=", 2)[1]
		}
		if arg == "--home" && i+1 < len(args) {
			return args[i+1]
		}
	}

	return app.DefaultNodeHome
}

// loadTelemetryPorts reads the desired metrics/health ports from config or
// environment variables, falling back to defaults when values are missing.
func loadTelemetryPorts(home string) (int, int) {
	metricsPort := defaultMetricsPort
	healthPort := defaultHealthPort

	configPath := filepath.Join(home, "config", "app.toml")
	if m, h, err := readTelemetryPortsFromConfig(configPath); err == nil {
		if m > 0 {
			metricsPort = m
		}
		if h > 0 {
			healthPort = h
		}
	}

	if env := os.Getenv("PAW_TELEMETRY_METRICS_PORT"); env != "" {
		if port := parsePort(env); port > 0 {
			metricsPort = port
		}
	}

	if env := os.Getenv("PAW_TELEMETRY_HEALTH_PORT"); env != "" {
		if port := parsePort(env); port > 0 {
			healthPort = port
		}
	}

	return metricsPort, healthPort
}

func readTelemetryPortsFromConfig(path string) (int, int, error) {
	v := viper.New()
	v.SetConfigType("toml")
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return 0, 0, err
	}

	return v.GetInt("telemetry.metrics-port"), v.GetInt("telemetry.health-port"), nil
}

func parsePort(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}

	port, err := strconv.Atoi(value)
	if err != nil || port <= 0 || port > 65535 {
		return 0
	}

	return port
}

// resolveRPCAddress chooses the RPC endpoint used by the health checker.
// It prefers PAW_RPC_ENDPOINT, then config/rpc.laddr, and finally a fallback.
func resolveRPCAddress(home string) string {
	if env := os.Getenv("PAW_RPC_ENDPOINT"); env != "" {
		return env
	}

	configPath := filepath.Join(home, "config", "config.toml")
	if hostPort, err := readRPCHostPort(configPath); err == nil && hostPort != "" {
		return fmt.Sprintf("http://%s", hostPort)
	}

	return defaultRPCAddress
}

func readRPCHostPort(path string) (string, error) {
	v := viper.New()
	v.SetConfigType("toml")
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return "", err
	}

	raw := v.GetString("rpc.laddr")
	if raw == "" {
		return "", fmt.Errorf("rpc.laddr not configured")
	}

	hostPort := raw
	if strings.Contains(raw, "://") {
		if parsed, err := url.Parse(raw); err == nil && parsed.Host != "" {
			hostPort = parsed.Host
		}
	}

	return sanitizeHostPort(hostPort), nil
}

func sanitizeHostPort(raw string) string {
	hostPort := strings.TrimSpace(raw)
	if hostPort == "" {
		return ""
	}

	host, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		return hostPort
	}

	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}

	return net.JoinHostPort(host, port)
}
