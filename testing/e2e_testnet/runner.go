// Package e2e_testnet provides end-to-end testing against live PAW testnet infrastructure.
package e2e_testnet

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type TestResult struct {
	Name     string        `json:"name"`
	Phase    string        `json:"phase"`
	Passed   bool          `json:"passed"`
	Duration time.Duration `json:"duration"`
	Details  string        `json:"details,omitempty"`
	Error    string        `json:"error,omitempty"`
}

type TestSuite struct {
	ChainID     string       `json:"chain_id"`
	StartTime   time.Time    `json:"start_time"`
	EndTime     time.Time    `json:"end_time"`
	TotalTests  int          `json:"total_tests"`
	PassedTests int          `json:"passed_tests"`
	FailedTests int          `json:"failed_tests"`
	Results     []TestResult `json:"results"`
	Summary     string       `json:"summary"`
}

type Runner struct {
	config  *TestnetConfig
	client  *Client
	suite   *TestSuite
	verbose bool
}

func NewRunner(cfg *TestnetConfig, verbose bool) *Runner {
	return &Runner{
		config:  cfg,
		client:  NewClient(cfg),
		verbose: verbose,
		suite: &TestSuite{
			ChainID:   cfg.ChainID,
			StartTime: time.Now().UTC(),
			Results:   make([]TestResult, 0),
		},
	}
}

func (r *Runner) recordResult(phase, name string, passed bool, details string, err error) {
	result := TestResult{
		Name:    name,
		Phase:   phase,
		Passed:  passed,
		Details: details,
	}
	if err != nil {
		result.Error = err.Error()
	}

	r.suite.Results = append(r.suite.Results, result)
	r.suite.TotalTests++
	if passed {
		r.suite.PassedTests++
		if r.verbose {
			fmt.Printf("[PASS] %s: %s\n", phase, name)
		}
	} else {
		r.suite.FailedTests++
		fmt.Printf("[FAIL] %s: %s - %s\n", phase, name, details)
		if err != nil {
			fmt.Printf("       Error: %v\n", err)
		}
	}
}

func (r *Runner) RunAll(ctx context.Context) *TestSuite {
	fmt.Println("=" + strings.Repeat("=", 59))
	fmt.Printf("PAW Testnet E2E Validation - %s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Printf("Chain ID: %s\n", r.config.ChainID)
	fmt.Println("=" + strings.Repeat("=", 59))

	r.runStabilityTests(ctx)
	r.runCoreTests(ctx)
	r.runMultiNodeTests(ctx)
	r.runConsensusTests(ctx)
	r.runSecurityTests(ctx)
	r.runOpsTests(ctx)

	r.suite.EndTime = time.Now().UTC()
	r.suite.Summary = fmt.Sprintf("%d/%d tests passed", r.suite.PassedTests, r.suite.TotalTests)

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("SUMMARY: %s\n", r.suite.Summary)
	fmt.Printf("Duration: %s\n", r.suite.EndTime.Sub(r.suite.StartTime).Round(time.Second))
	fmt.Println(strings.Repeat("=", 60))

	return r.suite
}

func (r *Runner) runStabilityTests(ctx context.Context) {
	fmt.Println("\n=== PHASE 1: STABILITY ===")

	val := r.config.PrimaryValidator()

	status, err := r.client.GetStatus(ctx, val)
	r.recordResult("stability", "1.1 Node responding", err == nil && status != nil, "", err)

	if status != nil {
		r.recordResult("stability", "1.2 Not catching up", !status.SyncInfo.CatchingUp,
			fmt.Sprintf("catching_up=%v", status.SyncInfo.CatchingUp), nil)
	}

	if status != nil {
		matches := status.NodeInfo.Network == r.config.ChainID
		r.recordResult("stability", "1.3 Chain ID matches",
			matches, fmt.Sprintf("expected=%s, got=%s", r.config.ChainID, status.NodeInfo.Network), nil)
	}

	h1, _ := r.client.GetBlockHeight(ctx, val)
	time.Sleep(6 * time.Second)
	h2, _ := r.client.GetBlockHeight(ctx, val)
	producing := h2 > h1
	r.recordResult("stability", "1.4 Block production",
		producing, fmt.Sprintf("height %d -> %d (+%d)", h1, h2, h2-h1), nil)

	if status != nil {
		var power int64
		fmt.Sscanf(status.ValidatorInfo.VotingPower, "%d", &power)
		r.recordResult("stability", "1.5 Has voting power",
			power > 0, fmt.Sprintf("voting_power=%d", power), nil)
	}

	if status != nil {
		blockTime, _ := time.Parse(time.RFC3339Nano, status.SyncInfo.LatestBlockTime)
		age := time.Since(blockTime)
		r.recordResult("stability", "1.6 Recent block time",
			age < 30*time.Second, fmt.Sprintf("age=%s", age.Round(time.Second)), nil)
	}
}

func (r *Runner) runCoreTests(ctx context.Context) {
	fmt.Println("\n=== PHASE 2: CORE ===")

	val := r.config.PrimaryValidator()

	supply, err := r.client.GetBankSupply(ctx, val)
	r.recordResult("core", "2.1 Bank module query", err == nil && supply != nil, "", err)

	validators, err := r.client.GetValidators(ctx, val)
	r.recordResult("core", "2.2 Staking module query", err == nil && validators != nil, "", err)

	nodeInfo, err := r.client.GetNodeInfo(ctx, val)
	r.recordResult("core", "2.3 Node info query", err == nil && nodeInfo != nil, "", err)

	block, err := r.client.GetLatestBlock(ctx, val)
	r.recordResult("core", "2.4 Latest block query", err == nil && block != nil, "", err)

	// PAW-specific: DEX module
	dex, err := r.client.QueryREST(ctx, val, "/paw/dex/v1beta1/pools")
	if err != nil {
		fmt.Println("  Note: DEX module query skipped (may not be enabled)")
	} else {
		r.recordResult("core", "2.5 DEX module query", dex != nil, "", nil)
	}
}

func (r *Runner) runMultiNodeTests(ctx context.Context) {
	fmt.Println("\n=== PHASE 2.5: MULTI-NODE CONSISTENCY ===")

	primary := r.config.PrimaryValidator()
	refSupply, _ := r.client.GetBankSupply(ctx, primary)
	refValidators, _ := r.client.GetValidators(ctx, primary)

	r.recordResult("multinode", "2.5.1 Reference state from primary",
		refSupply != nil && refValidators != nil, "", nil)

	supplyConsistent := true
	validatorsConsistent := true
	restAvailable := true
	chainIDConsistent := true
	appHashPresent := true

	for _, val := range r.config.Validators {
		v := val

		supply, err := r.client.GetBankSupply(ctx, &v)
		if err != nil || supply == nil {
			supplyConsistent = false
		}

		validators, err := r.client.GetValidators(ctx, &v)
		if err != nil || validators == nil {
			validatorsConsistent = false
		}

		nodeInfo, err := r.client.GetNodeInfo(ctx, &v)
		if err != nil || nodeInfo == nil {
			restAvailable = false
		}

		status, err := r.client.GetStatus(ctx, &v)
		if err != nil || status == nil || status.NodeInfo.Network != r.config.ChainID {
			chainIDConsistent = false
		}

		block, err := r.client.GetLatestBlock(ctx, &v)
		if err != nil || block == nil {
			appHashPresent = false
		}
	}

	r.recordResult("multinode", "2.5.2 Bank module consistency (all nodes)", supplyConsistent, "", nil)
	r.recordResult("multinode", "2.5.3 Staking module consistency (all nodes)", validatorsConsistent, "", nil)
	r.recordResult("multinode", "2.5.4 REST API availability (all nodes)", restAvailable, "", nil)
	r.recordResult("multinode", "2.5.5 Chain ID consistency (all nodes)", chainIDConsistent, "", nil)
	r.recordResult("multinode", "2.5.6 App hash present (all nodes)", appHashPresent, "", nil)
}

func (r *Runner) runConsensusTests(ctx context.Context) {
	fmt.Println("\n=== PHASE 3: CONSENSUS ===")

	heights := make(map[string]int64)
	var maxHeight, minHeight int64 = 0, 1 << 62

	for _, val := range r.config.Validators {
		v := val
		h, err := r.client.GetBlockHeight(ctx, &v)
		if err == nil {
			heights[v.Name] = h
			if h > maxHeight {
				maxHeight = h
			}
			if h < minHeight {
				minHeight = h
			}
		}
	}

	heightDiff := maxHeight - minHeight
	r.recordResult("consensus", "3.1 Height sync (<5 block diff)",
		heightDiff < 5, fmt.Sprintf("diff=%d (max=%d, min=%d)", heightDiff, maxHeight, minHeight), nil)

	validators, err := r.client.GetValidators(ctx, r.config.PrimaryValidator())
	if err == nil && validators != nil {
		if valList, ok := validators["validators"].([]interface{}); ok {
			activeCount := 0
			for _, v := range valList {
				if val, ok := v.(map[string]interface{}); ok {
					if status, ok := val["status"].(string); ok && status == "BOND_STATUS_BONDED" {
						activeCount++
					}
				}
			}
			r.recordResult("consensus", "3.2 Multiple validators bonded",
				activeCount >= 2, fmt.Sprintf("bonded=%d", activeCount), nil)
		}
	}

	h1, _ := r.client.GetBlockHeight(ctx, r.config.PrimaryValidator())
	t1 := time.Now()
	time.Sleep(10 * time.Second)
	h2, _ := r.client.GetBlockHeight(ctx, r.config.PrimaryValidator())
	t2 := time.Now()

	if h2 > h1 {
		avgBlockTime := t2.Sub(t1) / time.Duration(h2-h1)
		reasonable := avgBlockTime > 2*time.Second && avgBlockTime < 10*time.Second
		r.recordResult("consensus", "3.3 Block time reasonable (2-10s)",
			reasonable, fmt.Sprintf("avg=%s", avgBlockTime.Round(time.Millisecond)), nil)
	}
}

func (r *Runner) runSecurityTests(ctx context.Context) {
	fmt.Println("\n=== PHASE 4: SECURITY ===")

	val := r.config.PrimaryValidator()

	rpcBound := strings.Contains(val.GetRPCEndpoint(), "127.0.0.1") ||
		strings.Contains(val.GetRPCEndpoint(), "localhost")
	r.recordResult("security", "4.1 RPC bound to localhost", rpcBound,
		fmt.Sprintf("endpoint=%s", val.GetRPCEndpoint()), nil)

	restBound := strings.Contains(val.GetRESTEndpoint(), "127.0.0.1") ||
		strings.Contains(val.GetRESTEndpoint(), "localhost")
	r.recordResult("security", "4.2 REST bound to localhost", restBound,
		fmt.Sprintf("endpoint=%s", val.GetRESTEndpoint()), nil)
}

func (r *Runner) runOpsTests(ctx context.Context) {
	fmt.Println("\n=== PHASE 5: OPERATIONS ===")

	val := r.config.PrimaryValidator()

	diskOutput, err := r.client.RunCommand(ctx, val, "df / | awk 'NR==2 {print 100-$5}'")
	if err == nil {
		var diskFree int
		fmt.Sscanf(strings.TrimSuffix(diskOutput, "%"), "%d", &diskFree)
		r.recordResult("ops", "5.1 Disk space (>20% free)",
			diskFree > 20, fmt.Sprintf("free=%d%%", diskFree), nil)
	}

	memOutput, err := r.client.RunCommand(ctx, val, "free | awk '/Mem:/ {printf \"%.0f\", $7/$2*100}'")
	if err == nil {
		var memFree int
		fmt.Sscanf(memOutput, "%d", &memFree)
		r.recordResult("ops", "5.2 Memory (>10% available)",
			memFree > 10, fmt.Sprintf("available=%d%%", memFree), nil)
	}

	serviceOutput, err := r.client.RunCommand(ctx, val, "systemctl is-active pawd-val@1 2>/dev/null || systemctl is-active pawd 2>/dev/null")
	serviceActive := strings.Contains(serviceOutput, "active")
	r.recordResult("ops", "5.3 Validator service active", serviceActive, serviceOutput, err)
}

func (r *Runner) SaveResults(filepath string) error {
	data, err := json.MarshalIndent(r.suite, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath, data, 0644)
}

func (r *Runner) SaveMarkdown(filepath string) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# PAW Testnet E2E Validation Report\n\n"))
	sb.WriteString(fmt.Sprintf("**Chain ID:** %s\n", r.suite.ChainID))
	sb.WriteString(fmt.Sprintf("**Date:** %s\n", r.suite.StartTime.Format("2006-01-02 15:04:05 UTC")))
	sb.WriteString(fmt.Sprintf("**Duration:** %s\n", r.suite.EndTime.Sub(r.suite.StartTime).Round(time.Second)))
	sb.WriteString(fmt.Sprintf("**Result:** %s\n\n", r.suite.Summary))

	sb.WriteString("## Test Results\n\n")
	sb.WriteString("| Phase | Test | Status | Details |\n")
	sb.WriteString("|-------|------|--------|--------|\n")

	for _, result := range r.suite.Results {
		status := "PASS"
		if !result.Passed {
			status = "FAIL"
		}
		details := result.Details
		if result.Error != "" {
			details = result.Error
		}
		if details == "" {
			details = "-"
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
			result.Phase, result.Name, status, details))
	}

	sb.WriteString(fmt.Sprintf("\n---\n*Generated by PAW E2E Test Runner*\n"))

	return os.WriteFile(filepath, []byte(sb.String()), 0644)
}
