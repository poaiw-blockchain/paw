// Package e2e_testnet provides end-to-end testing against live PAW testnet infrastructure.
package e2e_testnet

import (
	"fmt"
	"os"
	"time"
)

// ValidatorConfig represents a single validator endpoint
type ValidatorConfig struct {
	Name     string
	Host     string
	RPCPort  int
	RESTPort int
	GRPCPort int
	P2PPort  int
	Home     string
}

// TestnetConfig holds the complete testnet configuration
type TestnetConfig struct {
	ChainID    string
	Validators []ValidatorConfig
	Faucet     FaucetConfig
	Explorer   ExplorerConfig
	Timeout    time.Duration
}

type FaucetConfig struct {
	Endpoint string
	Port     int
}

type ExplorerConfig struct {
	Endpoint string
	Port     int
}

// DefaultTestnetConfig returns the configuration for paw-mvp-1
func DefaultTestnetConfig() *TestnetConfig {
	return &TestnetConfig{
		ChainID: "paw-mvp-1",
		Validators: []ValidatorConfig{
			{
				Name:     "val1",
				Host:     "127.0.0.1",
				RPCPort:  11657,
				RESTPort: 11317,
				GRPCPort: 11090,
				P2PPort:  11656,
				Home:     "~/.paw-val1",
			},
			{
				Name:     "val2",
				Host:     "127.0.0.1",
				RPCPort:  11757,
				RESTPort: 11417,
				GRPCPort: 11190,
				P2PPort:  11756,
				Home:     "~/.paw-val2",
			},
			{
				Name:     "val3",
				Host:     "services-testnet",
				RPCPort:  11857,
				RESTPort: 11517,
				GRPCPort: 11290,
				P2PPort:  11856,
				Home:     "~/.paw-val3",
			},
			{
				Name:     "val4",
				Host:     "services-testnet",
				RPCPort:  11957,
				RESTPort: 11617,
				GRPCPort: 11390,
				P2PPort:  11956,
				Home:     "~/.paw-val4",
			},
		},
		Faucet: FaucetConfig{
			Endpoint: "http://127.0.0.1",
			Port:     8082,
		},
		Explorer: ExplorerConfig{
			Endpoint: "http://127.0.0.1",
			Port:     11080,
		},
		Timeout: 60 * time.Second,
	}
}

func (v *ValidatorConfig) GetRPCEndpoint() string {
	return fmt.Sprintf("http://%s:%d", v.Host, v.RPCPort)
}

func (v *ValidatorConfig) GetRESTEndpoint() string {
	return fmt.Sprintf("http://%s:%d", v.Host, v.RESTPort)
}

func (v *ValidatorConfig) GetGRPCEndpoint() string {
	return fmt.Sprintf("%s:%d", v.Host, v.GRPCPort)
}

func LoadConfigFromEnv(cfg *TestnetConfig) {
	if chainID := os.Getenv("PAW_CHAIN_ID"); chainID != "" {
		cfg.ChainID = chainID
	}
	if timeout := os.Getenv("PAW_TEST_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			cfg.Timeout = d
		}
	}
}

// LocalOnlyConfig returns config with only local validators (no SSH required)
func LocalOnlyConfig() *TestnetConfig {
	cfg := DefaultTestnetConfig()
	localVals := make([]ValidatorConfig, 0)
	for _, v := range cfg.Validators {
		if v.Host == "127.0.0.1" || v.Host == "localhost" {
			localVals = append(localVals, v)
		}
	}
	cfg.Validators = localVals
	return cfg
}

// DefaultOutputDir returns the default output directory for results
func DefaultOutputDir() string {
	home := os.Getenv("HOME")
	if home != "" {
		resultsDir := home + "/testnets/paw-mvp-1/results"
		if _, err := os.Stat(resultsDir); err == nil {
			return resultsDir
		}
		if err := os.MkdirAll(resultsDir, 0755); err == nil {
			return resultsDir
		}
	}
	return "."
}

func (c *TestnetConfig) PrimaryValidator() *ValidatorConfig {
	if len(c.Validators) > 0 {
		return &c.Validators[0]
	}
	return nil
}

func (c *TestnetConfig) GetValidatorByName(name string) *ValidatorConfig {
	for i := range c.Validators {
		if c.Validators[i].Name == name {
			return &c.Validators[i]
		}
	}
	return nil
}
