package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	cosmosaccount "github.com/ignite/cli/v29/ignite/pkg/cosmosaccount"
	cosmosclient "github.com/ignite/cli/v29/ignite/pkg/cosmosclient"

	dextypes "github.com/paw-chain/paw/x/dex/types"
)

// LoadTestConfig holds the configuration for the load test
type LoadTestConfig struct {
	RPCEndpoint    string
	APIEndpoint    string
	ChainID        string
	Duration       time.Duration
	Concurrency    int
	TxRate         int // transactions per second
	TestType       string
	ReportInterval time.Duration
	KeyringDir     string
	KeyringBackend string
	AccountNames   []string
	AddressPrefix  string
	CoinType       uint32
}

// LoadTestMetrics tracks performance metrics
type LoadTestMetrics struct {
	TotalTxSubmitted  uint64
	TotalTxSuccessful uint64
	TotalTxFailed     uint64
	TotalQueries      uint64
	TotalQueryFailed  uint64
	MinLatency        int64 // nanoseconds
	MaxLatency        int64
	TotalLatency      int64
	StartTime         time.Time
	EndTime           time.Time
	LatencyHistogram  map[int]uint64 // bucket (ms) -> count
	ErrorsByType      map[string]uint64
	mutex             sync.RWMutex
}

// NewLoadTestMetrics creates a new metrics tracker
func NewLoadTestMetrics() *LoadTestMetrics {
	return &LoadTestMetrics{
		MinLatency:       int64(^uint64(0) >> 1), // max int64
		MaxLatency:       0,
		StartTime:        time.Now(),
		LatencyHistogram: make(map[int]uint64),
		ErrorsByType:     make(map[string]uint64),
	}
}

var (
	defaultTokenPairs = []tokenPair{
		{In: "upaw", Out: "uatom"},
		{In: "upaw", Out: "uosmo"},
		{In: "uatom", Out: "uosmo"},
		{In: "upaw", Out: "uusdc"},
	}
)

const (
	defaultTxMinAmount = 1000
	defaultTxMaxAmount = 1000000
	defaultDenom       = "upaw"
)

const (
	bech32AccountPrefix             = "paw"
	bech32AccountPubPrefix          = "pawpub"
	bech32ValidatorPrefix           = "pawvaloper"
	bech32ValidatorPubPrefix        = "pawvaloperpub"
	bech32ConsensusPrefix           = "pawvalcons"
	bech32ConsensusPubPrefix        = "pawvalconspub"
	pawCoinType              uint32 = 118
)

type tokenPair struct {
	In  string
	Out string
}

type dexPool struct {
	ID string `json:"id"`
}

type poolsResponse struct {
	Pools []dexPool `json:"pools"`
}

type balancesResponse struct {
	Balances []json.RawMessage `json:"balances"`
}

// RecordTransaction records a transaction result
func (m *LoadTestMetrics) RecordTransaction(success bool, latency time.Duration, errType string) {
	atomic.AddUint64(&m.TotalTxSubmitted, 1)

	if success {
		atomic.AddUint64(&m.TotalTxSuccessful, 1)
	} else {
		atomic.AddUint64(&m.TotalTxFailed, 1)
		m.mutex.Lock()
		m.ErrorsByType[errType]++
		m.mutex.Unlock()
	}

	// Update latency stats
	latencyNs := latency.Nanoseconds()
	atomic.AddInt64(&m.TotalLatency, latencyNs)

	// Update min/max latency
	for {
		current := atomic.LoadInt64(&m.MinLatency)
		if latencyNs >= current || atomic.CompareAndSwapInt64(&m.MinLatency, current, latencyNs) {
			break
		}
	}

	for {
		current := atomic.LoadInt64(&m.MaxLatency)
		if latencyNs <= current || atomic.CompareAndSwapInt64(&m.MaxLatency, current, latencyNs) {
			break
		}
	}

	// Update histogram
	bucket := int(latency.Milliseconds() / 10) // 10ms buckets
	m.mutex.Lock()
	m.LatencyHistogram[bucket]++
	m.mutex.Unlock()
}

// RecordQuery records a query result
func (m *LoadTestMetrics) RecordQuery(success bool, latency time.Duration) {
	atomic.AddUint64(&m.TotalQueries, 1)
	if !success {
		atomic.AddUint64(&m.TotalQueryFailed, 1)
	}

	latencyNs := latency.Nanoseconds()
	atomic.AddInt64(&m.TotalLatency, latencyNs)
}

// GetReport generates a performance report
func (m *LoadTestMetrics) GetReport() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	duration := m.EndTime.Sub(m.StartTime).Seconds()
	avgLatency := time.Duration(0)
	if m.TotalTxSubmitted > 0 {
		avgLatency = time.Duration(m.TotalLatency / int64(m.TotalTxSubmitted))
	}

	tps := float64(m.TotalTxSuccessful) / duration

	return map[string]interface{}{
		"duration_seconds":     duration,
		"total_tx_submitted":   m.TotalTxSubmitted,
		"total_tx_successful":  m.TotalTxSuccessful,
		"total_tx_failed":      m.TotalTxFailed,
		"total_queries":        m.TotalQueries,
		"total_query_failed":   m.TotalQueryFailed,
		"transactions_per_sec": tps,
		"avg_latency_ms":       avgLatency.Milliseconds(),
		"min_latency_ms":       time.Duration(m.MinLatency).Milliseconds(),
		"max_latency_ms":       time.Duration(m.MaxLatency).Milliseconds(),
		"success_rate":         float64(m.TotalTxSuccessful) / float64(m.TotalTxSubmitted) * 100,
		"latency_histogram":    m.LatencyHistogram,
		"errors_by_type":       m.ErrorsByType,
	}
}

// LoadTester manages the load testing
type LoadTester struct {
	config      *LoadTestConfig
	metrics     *LoadTestMetrics
	ctx         context.Context
	cancel      context.CancelFunc
	httpClient  *http.Client
	apiEndpoint string
	addresses   []string
	tokenPairs  []tokenPair
	poolIDs     []uint64
	poolMu      sync.RWMutex
	rand        *rand.Rand
	randMu      sync.Mutex
	accountReg  cosmosaccount.Registry
	cosmosCli   cosmosclient.Client
	accounts    []*loadTestAccount
	accountIdx  uint64
	addressPref string
}

type loadTestAccount struct {
	name    string
	bech32  string
	address sdk.AccAddress
	account cosmosaccount.Account
}

// NewLoadTester creates a new load tester
func NewLoadTester(config *LoadTestConfig) (*LoadTester, error) {
	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)

	apiEndpoint := strings.TrimRight(config.APIEndpoint, "/")
	if apiEndpoint == "" {
		apiEndpoint = "http://localhost:1317"
	}

	if config.AddressPrefix == "" {
		config.AddressPrefix = bech32AccountPrefix
	}
	if config.CoinType == 0 {
		config.CoinType = pawCoinType
	}

	lt := &LoadTester{
		config:      config,
		metrics:     NewLoadTestMetrics(),
		ctx:         ctx,
		cancel:      cancel,
		httpClient:  &http.Client{Timeout: 15 * time.Second},
		apiEndpoint: apiEndpoint,
		addresses:   make([]string, 0, len(config.AccountNames)),
		tokenPairs:  append([]tokenPair(nil), defaultTokenPairs...),
		poolIDs:     make([]uint64, 0),
		rand:        rand.New(rand.NewSource(time.Now().UnixNano())),
		addressPref: config.AddressPrefix,
	}

	lt.configureSDK()

	if err := lt.initClients(); err != nil {
		cancel()
		return nil, fmt.Errorf("init clients: %w", err)
	}

	if err := lt.loadAccounts(); err != nil {
		cancel()
		return nil, fmt.Errorf("load accounts: %w", err)
	}

	if err := lt.refreshPoolIDs(); err != nil {
		log.Printf("load tester: failed to preload DEX pools: %v", err)
	}

	return lt, nil
}

func (lt *LoadTester) configureSDK() {
	config := sdk.GetConfig()
	if config.GetBech32AccountAddrPrefix() == lt.addressPref {
		return
	}

	config.SetBech32PrefixForAccount(bech32AccountPrefix, bech32AccountPubPrefix)
	config.SetBech32PrefixForValidator(bech32ValidatorPrefix, bech32ValidatorPubPrefix)
	config.SetBech32PrefixForConsensusNode(bech32ConsensusPrefix, bech32ConsensusPubPrefix)
	config.SetCoinType(lt.config.CoinType)
	config.Seal()
}

func (lt *LoadTester) initClients() error {
	backend := cosmosaccount.KeyringBackend(lt.config.KeyringBackend)
	if backend == "" {
		backend = cosmosaccount.KeyringTest
	}

	registry, err := cosmosaccount.New(
		cosmosaccount.WithHome(lt.config.KeyringDir),
		cosmosaccount.WithKeyringBackend(backend),
		cosmosaccount.WithBech32Prefix(lt.addressPref),
		cosmosaccount.WithCoinType(lt.config.CoinType),
	)
	if err != nil {
		return err
	}

	client, err := cosmosclient.New(
		lt.ctx,
		cosmosclient.WithNodeAddress(lt.config.RPCEndpoint),
		cosmosclient.WithKeyringBackend(backend),
		cosmosclient.WithKeyringDir(lt.config.KeyringDir),
		cosmosclient.WithBech32Prefix(lt.addressPref),
	)
	if err != nil {
		return err
	}

	lt.accountReg = registry
	lt.cosmosCli = client
	return nil
}

func (lt *LoadTester) loadAccounts() error {
	if len(lt.config.AccountNames) == 0 {
		return fmt.Errorf("no load test accounts configured")
	}

	lt.accounts = make([]*loadTestAccount, 0, len(lt.config.AccountNames))
	lt.addresses = lt.addresses[:0]

	for _, name := range lt.config.AccountNames {
		acc, err := lt.ensureAccount(name)
		if err != nil {
			return err
		}
		lt.accounts = append(lt.accounts, acc)
		lt.addresses = append(lt.addresses, acc.bech32)
	}

	return nil
}

func (lt *LoadTester) ensureAccount(name string) (*loadTestAccount, error) {
	account, err := lt.accountReg.GetByName(name)
	if err != nil {
		var notFound *cosmosaccount.AccountDoesNotExistError
		if errors.As(err, &notFound) {
			var mnemonic string
			account, mnemonic, err = lt.accountReg.Create(name)
			if err != nil {
				return nil, err
			}
			addr, addrErr := account.Address(lt.addressPref)
			if addrErr != nil {
				log.Printf("load tester: created new account %s. Please fund it before running high load", name)
			} else {
				log.Printf("load tester: created new account %s (%s). Please fund it before running high load", name, addr)
			}
			_ = mnemonic
		} else {
			return nil, err
		}
	}

	addrStr, err := account.Address(lt.addressPref)
	if err != nil {
		return nil, err
	}
	addr, err := sdk.AccAddressFromBech32(addrStr)
	if err != nil {
		return nil, err
	}

	return &loadTestAccount{
		name:    name,
		bech32:  addrStr,
		address: addr,
		account: account,
	}, nil
}

func (lt *LoadTester) nextAccount() *loadTestAccount {
	if len(lt.accounts) == 0 {
		return nil
	}
	idx := atomic.AddUint64(&lt.accountIdx, 1)
	return lt.accounts[idx%uint64(len(lt.accounts))]
}

// Run starts the load test
func (lt *LoadTester) Run() error {
	defer lt.cancel()

	log.Printf("Starting load test: %s", lt.config.TestType)
	log.Printf("Duration: %v, Concurrency: %d, Target TPS: %d",
		lt.config.Duration, lt.config.Concurrency, lt.config.TxRate)

	var wg sync.WaitGroup

	// Start metrics reporter
	go lt.reportMetrics()

	// Start workers
	for i := 0; i < lt.config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			lt.worker(workerID)
		}(i)
	}

	// Wait for completion
	wg.Wait()
	lt.metrics.EndTime = time.Now()

	return nil
}

// worker runs the actual load test operations
func (lt *LoadTester) worker(workerID int) {
	ticker := time.NewTicker(lt.tickInterval())
	defer ticker.Stop()

	for {
		select {
		case <-lt.ctx.Done():
			return
		case <-ticker.C:
			switch lt.config.TestType {
			case "transactions":
				lt.sendTransaction(workerID)
			case "queries":
				lt.performQuery(workerID)
			case "mixed":
				if workerID%2 == 0 {
					lt.sendTransaction(workerID)
				} else {
					lt.performQuery(workerID)
				}
			case "dex":
				lt.performDEXOperation(workerID)
			}
		}
	}
}

// sendTransaction sends a test transaction
func (lt *LoadTester) sendTransaction(workerID int) {
	start := time.Now()

	sender := lt.nextAccount()
	if sender == nil {
		lt.metrics.RecordTransaction(false, time.Since(start), "missing_sender")
		return
	}

	recipient := lt.randomAddress()
	for attempts := 0; attempts < 3 && (recipient == "" || recipient == sender.bech32); attempts++ {
		recipient = lt.randomAddress()
	}
	if recipient == "" || recipient == sender.bech32 {
		lt.metrics.RecordTransaction(false, time.Since(start), "missing_recipient")
		return
	}

	recAddr, err := sdk.AccAddressFromBech32(recipient)
	if err != nil {
		lt.metrics.RecordTransaction(false, time.Since(start), "recipient_decode_failed")
		return
	}

	amount := sdkmath.NewInt(int64(lt.randomAmount(defaultTxMinAmount, defaultTxMaxAmount)))
	msg := banktypes.NewMsgSend(sender.address, recAddr, sdk.NewCoins(sdk.NewCoin(defaultDenom, amount)))

	resp, err := lt.cosmosCli.BroadcastTx(lt.ctx, sender.account, msg)
	success := err == nil && resp.TxResponse != nil && resp.Code == 0
	errType := ""

	if err != nil {
		errType = "broadcast_error"
		log.Printf("worker %d: tx broadcast error from %s to %s: %v", workerID, sender.bech32, recipient, err)
	} else if resp.TxResponse == nil {
		errType = "empty_response"
		log.Printf("worker %d: tx broadcast returned empty response", workerID)
	} else if resp.Code != 0 {
		errType = fmt.Sprintf("abci_code_%d", resp.Code)
		log.Printf("worker %d: tx %s failed with code %d: %s", workerID, resp.TxHash, resp.Code, resp.RawLog)
	}

	lt.metrics.RecordTransaction(success, time.Since(start), errType)
}

// performQuery performs a test query
func (lt *LoadTester) performQuery(workerID int) {
	start := time.Now()

	address := lt.randomAddress()
	if address == "" {
		lt.metrics.RecordQuery(false, time.Since(start))
		return
	}

	req, err := http.NewRequestWithContext(lt.ctx, http.MethodGet, fmt.Sprintf("%s/cosmos/bank/v1beta1/balances/%s", lt.apiEndpoint, address), nil)
	if err != nil {
		log.Printf("worker %d: failed to create balance request: %v", workerID, err)
		lt.metrics.RecordQuery(false, time.Since(start))
		return
	}

	resp, err := lt.httpClient.Do(req)
	success := false

	if err != nil {
		log.Printf("worker %d: balance query HTTP error: %v", workerID, err)
	} else {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Printf("worker %d: balance query unexpected status %d", workerID, resp.StatusCode)
		} else {
			var payload balancesResponse
			if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
				log.Printf("worker %d: failed to decode balance response: %v", workerID, err)
			} else if len(payload.Balances) > 0 {
				success = true
			} else {
				log.Printf("worker %d: balance response empty for %s", workerID, address)
			}
		}
	}

	lt.metrics.RecordQuery(success, time.Since(start))
}

// performDEXOperation performs a DEX-specific operation
func (lt *LoadTester) performDEXOperation(workerID int) {
	start := time.Now()

	poolID, err := lt.randomPoolID()
	if err != nil {
		log.Printf("worker %d: unable to fetch DEX pool: %v", workerID, err)
		lt.metrics.RecordTransaction(false, time.Since(start), "no_pools")
		return
	}

	account := lt.nextAccount()
	if account == nil {
		lt.metrics.RecordTransaction(false, time.Since(start), "missing_trader")
		return
	}

	pair := lt.randomTokenPair()
	amount := sdkmath.NewInt(int64(lt.randomAmount(defaultTxMinAmount, defaultTxMaxAmount)))
	minAmount := amount.QuoRaw(2)
	if !minAmount.IsPositive() {
		minAmount = sdkmath.OneInt()
	}

	msg := &dextypes.MsgSwap{
		Trader:       account.bech32,
		PoolId:       poolID,
		TokenIn:      pair.In,
		TokenOut:     pair.Out,
		AmountIn:     amount,
		MinAmountOut: minAmount,
		Deadline:     time.Now().Add(2 * time.Minute).Unix(),
	}

	resp, err := lt.cosmosCli.BroadcastTx(lt.ctx, account.account, msg)
	success := err == nil && resp.TxResponse != nil && resp.Code == 0
	errType := ""

	if err != nil {
		errType = "broadcast_error"
		log.Printf("worker %d: swap broadcast error (pool %d): %v", workerID, poolID, err)
	} else if resp.TxResponse == nil {
		errType = "empty_response"
		log.Printf("worker %d: swap broadcast returned empty response", workerID)
	} else if resp.Code != 0 {
		errType = fmt.Sprintf("abci_code_%d", resp.Code)
		log.Printf("worker %d: swap tx %s failed with code %d: %s", workerID, resp.TxHash, resp.Code, resp.RawLog)
	}

	lt.metrics.RecordTransaction(success, time.Since(start), errType)
}

// reportMetrics periodically reports current metrics
func (lt *LoadTester) reportMetrics() {
	ticker := time.NewTicker(lt.config.ReportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-lt.ctx.Done():
			return
		case <-ticker.C:
			elapsed := time.Since(lt.metrics.StartTime).Seconds()
			tps := float64(lt.metrics.TotalTxSuccessful) / elapsed
			qps := float64(lt.metrics.TotalQueries) / elapsed

			log.Printf("[Progress] TX: %d (%.2f tps), Queries: %d (%.2f qps), Errors: %d",
				lt.metrics.TotalTxSubmitted,
				tps,
				lt.metrics.TotalQueries,
				qps,
				lt.metrics.TotalTxFailed,
			)
		}
	}
}

// SaveReport saves the final report to a file
func (lt *LoadTester) SaveReport(filename string) error {
	report := lt.metrics.GetReport()

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	log.Printf("Report saved to %s", filename)
	return nil
}

func (lt *LoadTester) randomAddress() string {
	if len(lt.addresses) == 0 {
		return ""
	}
	return lt.addresses[lt.randIntn(len(lt.addresses))]
}

func (lt *LoadTester) randomTokenPair() tokenPair {
	if len(lt.tokenPairs) == 0 {
		return tokenPair{In: defaultDenom, Out: defaultDenom}
	}
	return lt.tokenPairs[lt.randIntn(len(lt.tokenPairs))]
}

func (lt *LoadTester) randomAmount(min, max int) int {
	if max <= min {
		return min
	}
	return min + lt.randIntn(max-min)
}

func (lt *LoadTester) randIntn(n int) int {
	lt.randMu.Lock()
	defer lt.randMu.Unlock()
	if n <= 1 {
		return 0
	}
	return lt.rand.Intn(n)
}

func (lt *LoadTester) randomPoolID() (uint64, error) {
	lt.poolMu.RLock()
	hasPools := len(lt.poolIDs) > 0
	lt.poolMu.RUnlock()

	if !hasPools {
		if err := lt.refreshPoolIDs(); err != nil {
			return 0, err
		}
	}

	lt.poolMu.RLock()
	defer lt.poolMu.RUnlock()
	if len(lt.poolIDs) == 0 {
		return 0, fmt.Errorf("no DEX pools available")
	}

	return lt.poolIDs[lt.randIntn(len(lt.poolIDs))], nil
}

func (lt *LoadTester) refreshPoolIDs() error {
	req, err := http.NewRequestWithContext(lt.ctx, http.MethodGet, lt.joinAPI("/paw/dex/v1/pools?pagination.limit=100"), nil)
	if err != nil {
		return err
	}

	resp, err := lt.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pools query status %d: %s", resp.StatusCode, string(body))
	}

	var payload poolsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return err
	}

	ids := make([]uint64, 0, len(payload.Pools))
	for _, pool := range payload.Pools {
		if pool.ID == "" {
			continue
		}
		id, err := strconv.ParseUint(pool.ID, 10, 64)
		if err != nil {
			log.Printf("load tester: skipping pool with invalid id %q: %v", pool.ID, err)
			continue
		}
		ids = append(ids, id)
	}

	lt.poolMu.Lock()
	defer lt.poolMu.Unlock()
	lt.poolIDs = ids

	return nil
}

func (lt *LoadTester) joinAPI(path string) string {
	if path == "" {
		return lt.apiEndpoint
	}
	if strings.HasPrefix(path, "/") {
		return lt.apiEndpoint + path
	}
	return lt.apiEndpoint + "/" + path
}

func (lt *LoadTester) tickInterval() time.Duration {
	rate := lt.config.TxRate
	if rate <= 0 {
		rate = 1
	}

	concurrency := lt.config.Concurrency
	if concurrency <= 0 {
		concurrency = 1
	}

	interval := time.Duration(float64(time.Second) * float64(concurrency) / float64(rate))
	if interval <= 0 {
		return time.Millisecond
	}

	return interval
}

func parseAccountNames(raw, prefix string, fallback int) []string {
	var names []string
	for _, part := range strings.Split(raw, ",") {
		name := strings.TrimSpace(part)
		if name != "" {
			names = append(names, name)
		}
	}

	if len(names) > 0 {
		return names
	}

	if fallback <= 0 {
		fallback = 1
	}
	if prefix == "" {
		prefix = "loadtest"
	}

	for i := 0; i < fallback; i++ {
		names = append(names, fmt.Sprintf("%s-%d", prefix, i))
	}

	return names
}

func main() {
	// Parse command-line flags
	rpcEndpoint := flag.String("rpc", "http://localhost:26657", "RPC endpoint")
	apiEndpoint := flag.String("api", "http://localhost:1317", "API endpoint")
	chainID := flag.String("chain-id", "paw-testnet-1", "Chain ID")
	duration := flag.Duration("duration", 5*time.Minute, "Test duration")
	concurrency := flag.Int("concurrency", 10, "Number of concurrent workers")
	txRate := flag.Int("rate", 100, "Target transactions per second")
	testType := flag.String("type", "transactions", "Test type: transactions, queries, mixed, dex")
	reportInterval := flag.Duration("report-interval", 10*time.Second, "Metrics reporting interval")
	outputFile := flag.String("output", "load-test-report.json", "Output file for report")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	defaultKeyringDir := filepath.Join(homeDir, ".paw-loadtest")
	keyringDir := flag.String("keyring-dir", defaultKeyringDir, "Directory for load test keyring")
	keyringBackend := flag.String("keyring-backend", "test", "Keyring backend to use (test|os|memory)")
	accountsFlag := flag.String("accounts", "", "Comma-separated key names to use for load testing")
	numAccounts := flag.Int("num-accounts", 4, "Number of accounts to auto-create when --accounts is empty")
	accountPrefix := flag.String("account-prefix", "loadtest", "Prefix for auto-created load test accounts")

	flag.Parse()

	accountNames := parseAccountNames(*accountsFlag, *accountPrefix, *numAccounts)

	config := &LoadTestConfig{
		RPCEndpoint:    *rpcEndpoint,
		APIEndpoint:    *apiEndpoint,
		ChainID:        *chainID,
		Duration:       *duration,
		Concurrency:    *concurrency,
		TxRate:         *txRate,
		TestType:       *testType,
		ReportInterval: *reportInterval,
		KeyringDir:     *keyringDir,
		KeyringBackend: *keyringBackend,
		AccountNames:   accountNames,
		AddressPrefix:  bech32AccountPrefix,
		CoinType:       pawCoinType,
	}

	tester, err := NewLoadTester(config)
	if err != nil {
		log.Fatalf("failed to initialize load tester: %v", err)
	}

	log.Println("PAW Blockchain Load Tester")
	log.Println("===========================")

	if err := tester.Run(); err != nil {
		log.Fatalf("Load test failed: %v", err)
	}

	// Print final report
	report := tester.metrics.GetReport()
	fmt.Println("\n=== Load Test Results ===")
	fmt.Printf("Duration: %.2f seconds\n", report["duration_seconds"])
	fmt.Printf("Total Transactions: %d\n", report["total_tx_submitted"])
	fmt.Printf("Successful: %d\n", report["total_tx_successful"])
	fmt.Printf("Failed: %d\n", report["total_tx_failed"])
	fmt.Printf("TPS: %.2f\n", report["transactions_per_sec"])
	fmt.Printf("Success Rate: %.2f%%\n", report["success_rate"])
	fmt.Printf("Avg Latency: %d ms\n", report["avg_latency_ms"])
	fmt.Printf("Min Latency: %d ms\n", report["min_latency_ms"])
	fmt.Printf("Max Latency: %d ms\n", report["max_latency_ms"])

	// Save detailed report
	if err := tester.SaveReport(*outputFile); err != nil {
		log.Printf("Failed to save report: %v", err)
	}
}
