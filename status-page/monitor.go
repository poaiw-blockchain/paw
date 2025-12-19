package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type componentKind string

const (
	componentHTTP componentKind = "http"
	componentGRPC componentKind = "grpc"
)

// ComponentTarget describes a monitored dependency.
type ComponentTarget struct {
	Name        string
	Kind        componentKind
	Endpoint    string
	Description string
}

// ComponentStatus captures the most recent observation.
type ComponentStatus struct {
	Name        string    `json:"name"`
	Kind        string    `json:"kind"`
	Endpoint    string    `json:"endpoint"`
	Healthy     bool      `json:"healthy"`
	StatusCode  int       `json:"status_code"`
	LatencyMs   float64   `json:"latency_ms"`
	Message     string    `json:"message"`
	LastChecked time.Time `json:"last_checked"`
	UptimePct   float64   `json:"uptime_pct"`
}

// observation is stored in memory for uptime math.
type observation struct {
	healthy bool
}

// Monitor performs scheduled checks against all components.
type Monitor struct {
	cfg         Config
	targets     []ComponentTarget
	client      *http.Client
	grpcTimeout time.Duration
	mu          sync.RWMutex
	statuses    map[string]ComponentStatus
	history     map[string][]observation
}

func NewMonitor(cfg Config) *Monitor {
	timeout := cfg.HTTPTimeout
	if timeout == 0 {
		timeout = 6 * time.Second
	}

	targets := []ComponentTarget{
		{
			Name:        "RPC",
			Kind:        componentHTTP,
			Endpoint:    strings.TrimSuffix(cfg.RPCEndpoint, "/") + "/health",
			Description: "CometBFT RPC health probe",
		},
		{
			Name:        "REST",
			Kind:        componentHTTP,
			Endpoint:    strings.TrimSuffix(cfg.RESTEndpoint, "/") + "/cosmos/base/tendermint/v1beta1/node_info",
			Description: "Cosmos SDK REST API",
		},
		{
			Name:        "gRPC",
			Kind:        componentGRPC,
			Endpoint:    cfg.GRPCEndpoint,
			Description: "Cosmos gRPC health",
		},
		{
			Name:        "Explorer",
			Kind:        componentHTTP,
			Endpoint:    cfg.ExplorerURL,
			Description: "Block explorer frontend",
		},
		{
			Name:        "Faucet",
			Kind:        componentHTTP,
			Endpoint:    cfg.FaucetHealth,
			Description: "Faucet health endpoint",
		},
		{
			Name:        "Metrics",
			Kind:        componentHTTP,
			Endpoint:    cfg.MetricsURL,
			Description: "Prometheus/telemetry endpoint",
		},
	}

	return &Monitor{
		cfg:     cfg,
		targets: targets,
		client: &http.Client{
			Timeout: timeout,
		},
		grpcTimeout: 5 * time.Second,
		statuses:    make(map[string]ComponentStatus),
		history:     make(map[string][]observation),
	}
}

func (m *Monitor) Start(ctx context.Context) {
	m.runChecks()

	t := time.NewTicker(m.cfg.CheckInterval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			m.runChecks()
		}
	}
}

func (m *Monitor) runChecks() {
	wg := sync.WaitGroup{}

	for _, target := range m.targets {
		t := target
		wg.Add(1)
		go func() {
			defer wg.Done()
			status := m.checkTarget(t)
			m.recordStatus(status)
		}()
	}

	wg.Wait()
}

func (m *Monitor) checkTarget(target ComponentTarget) ComponentStatus {
	switch target.Kind {
	case componentHTTP:
		return m.checkHTTP(target)
	case componentGRPC:
		return m.checkGRPC(target)
	default:
		return ComponentStatus{
			Name:        target.Name,
			Kind:        string(target.Kind),
			Endpoint:    target.Endpoint,
			Healthy:     false,
			Message:     "unknown component type",
			LastChecked: time.Now(),
		}
	}
}

func (m *Monitor) checkHTTP(target ComponentTarget) ComponentStatus {
	start := time.Now()
	resp, err := m.client.Get(target.Endpoint)
	if err != nil {
		return m.errorStatus(target, err, 0, time.Since(start))
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	ok := resp.StatusCode >= 200 && resp.StatusCode < 400
	msg := http.StatusText(resp.StatusCode)
	if !ok {
		msg = fmt.Sprintf("status %d", resp.StatusCode)
	}

	return ComponentStatus{
		Name:        target.Name,
		Kind:        string(target.Kind),
		Endpoint:    target.Endpoint,
		Healthy:     ok,
		StatusCode:  resp.StatusCode,
		LatencyMs:   float64(time.Since(start).Milliseconds()),
		Message:     msg,
		LastChecked: time.Now(),
	}
}

func (m *Monitor) checkGRPC(target ComponentTarget) ComponentStatus {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), m.grpcTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, target.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return m.errorStatus(target, err, 0, time.Since(start))
	}
	defer conn.Close()

	client := healthpb.NewHealthClient(conn)
	healthCtx, healthCancel := context.WithTimeout(context.Background(), m.grpcTimeout)
	defer healthCancel()
	resp, err := client.Check(healthCtx, &healthpb.HealthCheckRequest{})
	if err != nil {
		return m.errorStatus(target, err, 0, time.Since(start))
	}

	ok := resp.GetStatus() == healthpb.HealthCheckResponse_SERVING
	code := http.StatusOK
	if !ok {
		code = http.StatusServiceUnavailable
	}

	return ComponentStatus{
		Name:        target.Name,
		Kind:        string(target.Kind),
		Endpoint:    target.Endpoint,
		Healthy:     ok,
		StatusCode:  code,
		LatencyMs:   float64(time.Since(start).Milliseconds()),
		Message:     resp.GetStatus().String(),
		LastChecked: time.Now(),
	}
}

func (m *Monitor) errorStatus(target ComponentTarget, err error, code int, duration time.Duration) ComponentStatus {
	msg := err.Error()
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		msg = "timeout"
	}

	return ComponentStatus{
		Name:        target.Name,
		Kind:        string(target.Kind),
		Endpoint:    target.Endpoint,
		Healthy:     false,
		StatusCode:  code,
		LatencyMs:   float64(duration.Milliseconds()),
		Message:     msg,
		LastChecked: time.Now(),
	}
}

func (m *Monitor) recordStatus(status ComponentStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()

	history := append(m.history[status.Name], observation{healthy: status.Healthy})
	if len(history) > 500 {
		history = history[len(history)-500:]
	}
	m.history[status.Name] = history

	successes := 0
	for _, h := range history {
		if h.healthy {
			successes++
		}
	}
	status.UptimePct = float64(successes) / float64(len(history)) * 100

	m.statuses[status.Name] = status
}

func (m *Monitor) Snapshot() (overall string, components []ComponentStatus, updatedAt time.Time) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	components = make([]ComponentStatus, 0, len(m.statuses))
	overall = "operational"
	for _, s := range m.statuses {
		if !s.Healthy && overall != "down" {
			overall = "degraded"
		}
		if !s.Healthy && s.StatusCode >= 500 {
			overall = "down"
		}
		components = append(components, s)
		if s.LastChecked.After(updatedAt) {
			updatedAt = s.LastChecked
		}
	}

	return overall, components, updatedAt
}
