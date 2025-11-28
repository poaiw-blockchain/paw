package rpc

import (
	"context"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid configuration",
			config: Config{
				RPCURL:         "http://localhost:26657",
				Timeout:        30 * time.Second,
				MaxRetries:     3,
				RequestsPerSec: 10,
			},
			wantErr: false,
		},
		{
			name: "empty RPC URL",
			config: Config{
				RPCURL: "",
			},
			wantErr: true,
		},
		{
			name: "defaults applied",
			config: Config{
				RPCURL: "http://localhost:26657",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewClient() returned nil client without error")
			}
			if client != nil {
				client.Close()
			}
		})
	}
}

func TestClient_GetChainHeight(t *testing.T) {
	// This test requires a running blockchain node
	// Skip in CI/CD environments
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClient(Config{
		RPCURL:         "http://localhost:26657",
		Timeout:        10 * time.Second,
		MaxRetries:     3,
		RequestsPerSec: 10,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	height, err := client.GetChainHeight(ctx)
	if err != nil {
		t.Fatalf("GetChainHeight() error = %v", err)
	}

	if height < 0 {
		t.Errorf("GetChainHeight() returned invalid height: %d", height)
	}

	t.Logf("Current chain height: %d", height)
}

func TestClient_GetBlock(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClient(Config{
		RPCURL:         "http://localhost:26657",
		Timeout:        10 * time.Second,
		MaxRetries:     3,
		RequestsPerSec: 10,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Get latest height first
	height, err := client.GetChainHeight(ctx)
	if err != nil {
		t.Fatalf("GetChainHeight() error = %v", err)
	}

	// Fetch a recent block
	block, err := client.GetBlock(ctx, height)
	if err != nil {
		t.Fatalf("GetBlock() error = %v", err)
	}

	if block == nil {
		t.Fatal("GetBlock() returned nil block")
	}

	t.Logf("Block height: %s, Hash: %s", block.Result.Block.Header.Height, block.Result.BlockID.Hash)
}

func TestClient_GetBlockBatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClient(Config{
		RPCURL:         "http://localhost:26657",
		Timeout:        30 * time.Second,
		MaxRetries:     3,
		RequestsPerSec: 10,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Get latest height
	height, err := client.GetChainHeight(ctx)
	if err != nil {
		t.Fatalf("GetChainHeight() error = %v", err)
	}

	// Fetch last 10 blocks
	startHeight := height - 10
	if startHeight < 1 {
		startHeight = 1
	}

	blocks, err := client.GetBlockBatch(ctx, startHeight, height)
	if err != nil {
		t.Fatalf("GetBlockBatch() error = %v", err)
	}

	expectedCount := height - startHeight + 1
	if int64(len(blocks)) != expectedCount {
		t.Errorf("GetBlockBatch() returned %d blocks, expected %d", len(blocks), expectedCount)
	}

	t.Logf("Successfully fetched %d blocks in batch", len(blocks))
}

func TestClient_Health(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClient(Config{
		RPCURL:         "http://localhost:26657",
		Timeout:        10 * time.Second,
		MaxRetries:     3,
		RequestsPerSec: 10,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	if err := client.Health(ctx); err != nil {
		t.Errorf("Health() error = %v", err)
	}
}

func BenchmarkClient_GetBlock(b *testing.B) {
	client, err := NewClient(Config{
		RPCURL:         "http://localhost:26657",
		Timeout:        30 * time.Second,
		MaxRetries:     3,
		RequestsPerSec: 100,
	})
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	height, err := client.GetChainHeight(ctx)
	if err != nil {
		b.Fatalf("GetChainHeight() error = %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.GetBlock(ctx, height)
		if err != nil {
			b.Errorf("GetBlock() error = %v", err)
		}
	}
}
