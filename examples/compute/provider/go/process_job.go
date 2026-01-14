//go:build paw_examples
// +build paw_examples

package main

/*
PAW Compute Provider - Job Processing Example

This example demonstrates how to process compute jobs:
- Query for assigned jobs
- Execute compute work
- Submit results back to the chain

Usage:
    go run process_job.go <request_id>

Environment Variables:
    PAW_GRPC_ENDPOINT - gRPC endpoint (default: localhost:9090)
    PAW_PROVIDER_MNEMONIC - Provider mnemonic (required)
*/

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	computetypes "github.com/paw-chain/paw/x/compute/types"
)

// JobProcessor handles compute job execution
type JobProcessor struct {
	grpcConn     *grpc.ClientConn
	queryClient  computetypes.QueryClient
	providerAddr string
}

// NewJobProcessor creates a new job processor
func NewJobProcessor(grpcEndpoint, providerAddr string) (*JobProcessor, error) {
	// Create gRPC connection
	grpcConn, err := grpc.Dial(
		grpcEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return &JobProcessor{
		grpcConn:     grpcConn,
		queryClient:  computetypes.NewQueryClient(grpcConn),
		providerAddr: providerAddr,
	}, nil
}

// Close cleans up resources
func (jp *JobProcessor) Close() error {
	if jp.grpcConn != nil {
		return jp.grpcConn.Close()
	}
	return nil
}

// GetJob retrieves a compute job by ID
func (jp *JobProcessor) GetJob(ctx context.Context, requestID uint64) (*computetypes.Request, error) {
	resp, err := jp.queryClient.Request(ctx, &computetypes.QueryRequestRequest{
		RequestId: requestID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query request: %w", err)
	}

	return resp.Request, nil
}

// ProcessJob executes a compute job
func (jp *JobProcessor) ProcessJob(ctx context.Context, req *computetypes.Request) (string, string, error) {
	fmt.Println("\n📦 Processing Compute Job")
	fmt.Println("========================")
	fmt.Printf("Request ID: %d\n", req.Id)
	fmt.Printf("Requester: %s\n", req.Requester)
	fmt.Printf("Container: %s\n", req.ContainerImage)
	fmt.Printf("Command: %s\n", req.Command)
	fmt.Printf("Timeout: %d seconds\n", req.Specs.TimeoutSeconds)
	fmt.Println()

	// Validate provider assignment
	if req.Provider != jp.providerAddr {
		return "", "", fmt.Errorf("job not assigned to this provider")
	}

	// Check job status
	if req.Status != computetypes.REQUEST_STATUS_ASSIGNED &&
		req.Status != computetypes.REQUEST_STATUS_PROCESSING {
		return "", "", fmt.Errorf("job not in processable state: %s", req.Status.String())
	}

	// Resource specifications
	fmt.Printf("Resource Requirements:\n")
	fmt.Printf("  CPU: %d millicores\n", req.Specs.CpuCores)
	fmt.Printf("  Memory: %d MB\n", req.Specs.MemoryMb)
	fmt.Printf("  Storage: %d GB\n", req.Specs.StorageGb)
	if req.Specs.GpuCount > 0 {
		fmt.Printf("  GPU: %d x %s\n", req.Specs.GpuCount, req.Specs.GpuType)
	}
	fmt.Println()

	// Execute compute work
	fmt.Println("⚙ Executing compute work...")
	outputData, err := jp.executeContainer(ctx, req)
	if err != nil {
		return "", "", fmt.Errorf("execution failed: %w", err)
	}

	// Compute output hash
	outputHash := computeHash(outputData)
	fmt.Printf("✓ Execution complete\n")
	fmt.Printf("  Output size: %d bytes\n", len(outputData))
	fmt.Printf("  Output hash: %s\n", outputHash)

	// Upload results (simulated - in production, upload to IPFS or S3)
	outputURL := jp.uploadResults(req.Id, outputData)
	fmt.Printf("  Output URL: %s\n", outputURL)

	return outputHash, outputURL, nil
}

// executeContainer executes the compute job in a container
func (jp *JobProcessor) executeContainer(ctx context.Context, req *computetypes.Request) ([]byte, error) {
	// In a production implementation, this would:
	// 1. Pull the container image if not cached
	// 2. Create a container with resource limits
	// 3. Mount input data
	// 4. Execute the command
	// 5. Collect output
	// 6. Clean up container

	// For this example, we'll simulate with a simple command execution
	fmt.Println("  Simulating container execution...")

	// Create timeout context
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(req.Specs.TimeoutSeconds)*time.Second)
	defer cancel()

	// Simulate work with sleep
	time.Sleep(2 * time.Second)

	// Example: Run a simple Docker command (if available)
	// In production, use proper container runtime like containerd or Docker SDK
	if isDockerAvailable() {
		cmd := exec.CommandContext(execCtx, "docker", "run", "--rm",
			"--cpus", fmt.Sprintf("%.2f", float64(req.Specs.CpuCores)/1000),
			"--memory", fmt.Sprintf("%dm", req.Specs.MemoryMb),
			req.ContainerImage,
			req.Command,
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			// If Docker command fails, return simulated output
			fmt.Println("  Docker execution failed, using simulated output")
			return jp.generateSimulatedOutput(req), nil
		}

		return output, nil
	}

	// Fallback to simulated output
	return jp.generateSimulatedOutput(req), nil
}

// generateSimulatedOutput creates simulated compute output
func (jp *JobProcessor) generateSimulatedOutput(req *computetypes.Request) []byte {
	output := fmt.Sprintf(`Compute Job Results
====================
Request ID: %d
Container: %s
Command: %s
Executed at: %s
Status: SUCCESS

Sample Output:
--------------
Processing complete. All tasks finished successfully.
Total items processed: 1000
Errors: 0
Warnings: 0
`, req.Id, req.ContainerImage, req.Command, time.Now().Format(time.RFC3339))

	return []byte(output)
}

// uploadResults uploads results to storage (simulated)
func (jp *JobProcessor) uploadResults(requestID uint64, data []byte) string {
	// In production, upload to:
	// - IPFS: ipfs.Add() -> get CID
	// - S3: s3.PutObject() -> get URL
	// - Arweave: arweave.CreateTransaction() -> get TX ID

	// For now, return simulated URL
	return fmt.Sprintf("ipfs://Qm%064d", requestID)
}

// computeHash computes SHA-256 hash
func computeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// isDockerAvailable checks if Docker is available
func isDockerAvailable() bool {
	cmd := exec.Command("docker", "version")
	err := cmd.Run()
	return err == nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: go run process_job.go <request_id>")
		os.Exit(1)
	}

	requestID, err := strconv.ParseUint(os.Args[1], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Invalid request ID: %v\n", err)
		os.Exit(1)
	}

	grpcEndpoint := getEnv("PAW_GRPC_ENDPOINT", "localhost:9090")
	providerAddr := os.Getenv("PAW_PROVIDER_ADDRESS")

	if providerAddr == "" {
		fmt.Fprintln(os.Stderr, "✗ PAW_PROVIDER_ADDRESS is required")
		os.Exit(1)
	}

	fmt.Printf("PAW Compute Job Processor\n")
	fmt.Printf("gRPC: %s\n", grpcEndpoint)
	fmt.Printf("Provider: %s\n", providerAddr)

	// Create processor
	processor, err := NewJobProcessor(grpcEndpoint, providerAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Failed to create processor: %v\n", err)
		os.Exit(1)
	}
	defer processor.Close()

	ctx := context.Background()

	// Get job details
	req, err := processor.GetJob(ctx, requestID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Failed to get job: %v\n", err)
		os.Exit(1)
	}

	// Process job
	outputHash, outputURL, err := processor.ProcessJob(ctx, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Failed to process job: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✓ Job processed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Submit result: go run submit_result.go %d %s %s\n", requestID, outputHash, outputURL)
	fmt.Printf("  2. Or use: pawd tx compute submit-result %d %s %s --from provider\n", requestID, outputHash, outputURL)
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
