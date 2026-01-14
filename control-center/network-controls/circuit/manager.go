package circuit

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricsOnce sync.Once

	sharedCircuitBreakerStatus *prometheus.GaugeVec
	sharedStateTransitions     *prometheus.CounterVec
	sharedAutoResumes          prometheus.Counter
)

// Manager handles circuit breaker operations and auto-resume logic
type Manager struct {
	registry       *CircuitBreakerRegistry
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	autoResumeFunc func(module, subModule string) error

	// Metrics
	circuitBreakerStatus *prometheus.GaugeVec
	stateTransitions     *prometheus.CounterVec
	autoResumes          prometheus.Counter
}

// NewManager creates a new circuit breaker manager
func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	metricsOnce.Do(func() {
		sharedCircuitBreakerStatus = promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "circuit_breaker_status",
				Help: "Current status of circuit breakers (0=closed, 1=open, 2=half-open)",
			},
			[]string{"module", "submodule"},
		)
		sharedStateTransitions = promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "circuit_breaker_transitions_total",
				Help: "Total number of circuit breaker state transitions",
			},
			[]string{"module", "submodule", "from", "to"},
		)
		sharedAutoResumes = promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "circuit_breaker_auto_resumes_total",
				Help: "Total number of automatic circuit breaker resumes",
			},
		)
	})

	return &Manager{
		registry:             NewCircuitBreakerRegistry(),
		ctx:                  ctx,
		cancel:               cancel,
		circuitBreakerStatus: sharedCircuitBreakerStatus,
		stateTransitions:     sharedStateTransitions,
		autoResumes:          sharedAutoResumes,
	}
}

// Start starts the manager's background tasks
func (m *Manager) Start() {
	m.wg.Add(1)
	go m.autoResumeLoop()
}

// Stop stops the manager gracefully
func (m *Manager) Stop() {
	m.cancel()
	m.wg.Wait()
}

// RegisterCircuitBreaker registers a new circuit breaker
func (m *Manager) RegisterCircuitBreaker(module, subModule string) error {
	if err := m.registry.Register(module, subModule); err != nil {
		return err
	}

	// Initialize metrics
	m.updateMetrics(module, subModule, StatusClosed)
	log.Printf("Circuit breaker registered: %s:%s", module, subModule)
	return nil
}

// PauseModule pauses a module (opens circuit breaker)
func (m *Manager) PauseModule(module, subModule, actor, reason string, autoResumeAfter *time.Duration) error {
	if err := m.registry.Open(module, subModule, actor, reason, autoResumeAfter); err != nil {
		return err
	}

	m.updateMetrics(module, subModule, StatusOpen)
	m.stateTransitions.WithLabelValues(module, subModule, string(StatusClosed), string(StatusOpen)).Inc()

	log.Printf("Circuit breaker opened: module=%s, submodule=%s, actor=%s, reason=%s",
		module, subModule, actor, reason)

	return nil
}

// ResumeModule resumes a module (closes circuit breaker)
func (m *Manager) ResumeModule(module, subModule, actor, reason string) error {
	state, err := m.registry.GetState(module, subModule)
	if err != nil {
		return err
	}

	oldStatus := state.Status

	if err := m.registry.Close(module, subModule, actor, reason); err != nil {
		return err
	}

	m.updateMetrics(module, subModule, StatusClosed)
	m.stateTransitions.WithLabelValues(module, subModule, string(oldStatus), string(StatusClosed)).Inc()

	log.Printf("Circuit breaker closed: module=%s, submodule=%s, actor=%s, reason=%s",
		module, subModule, actor, reason)

	return nil
}

// SetHalfOpen sets a circuit breaker to half-open state
func (m *Manager) SetHalfOpen(module, subModule, actor, reason string) error {
	if err := m.registry.HalfOpen(module, subModule, actor, reason); err != nil {
		return err
	}

	m.updateMetrics(module, subModule, StatusHalfOpen)
	m.stateTransitions.WithLabelValues(module, subModule, string(StatusOpen), string(StatusHalfOpen)).Inc()

	log.Printf("Circuit breaker set to half-open: module=%s, submodule=%s, actor=%s",
		module, subModule, actor)

	return nil
}

// IsBlocked checks if operations are blocked for a module
func (m *Manager) IsBlocked(module, subModule string) bool {
	return m.registry.IsOpen(module, subModule)
}

// GetState retrieves the state of a circuit breaker
func (m *Manager) GetState(module, subModule string) (*CircuitBreakerState, error) {
	return m.registry.GetState(module, subModule)
}

// GetAllStates returns all circuit breaker states
func (m *Manager) GetAllStates() map[string]*CircuitBreakerState {
	return m.registry.GetAllStates()
}

// SetMetadata sets metadata for a circuit breaker
func (m *Manager) SetMetadata(module, subModule string, metadata map[string]interface{}) error {
	return m.registry.SetMetadata(module, subModule, metadata)
}

// SetAutoResumeCallback sets a callback function to be called when auto-resuming
func (m *Manager) SetAutoResumeCallback(fn func(module, subModule string) error) {
	m.autoResumeFunc = fn
}

// autoResumeLoop periodically checks for circuit breakers that should auto-resume
func (m *Manager) autoResumeLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			log.Println("Circuit breaker auto-resume loop stopped")
			return
		case <-ticker.C:
			m.checkAutoResume()
		}
	}
}

// checkAutoResume checks and processes auto-resume timers
func (m *Manager) checkAutoResume() {
	resumed := m.registry.CheckAutoResume()

	for _, key := range resumed {
		m.autoResumes.Inc()
		log.Printf("Auto-resumed circuit breaker: %s", key)

		// Update metrics
		// Parse key to get module and submodule
		module, subModule := parseKey(key)
		m.updateMetrics(module, subModule, StatusClosed)
		m.stateTransitions.WithLabelValues(module, subModule, string(StatusOpen), string(StatusClosed)).Inc()

		// Call callback if set
		if m.autoResumeFunc != nil {
			if err := m.autoResumeFunc(module, subModule); err != nil {
				log.Printf("Auto-resume callback failed for %s: %v", key, err)
			}
		}
	}
}

// updateMetrics updates Prometheus metrics for a circuit breaker
func (m *Manager) updateMetrics(module, subModule string, status CircuitBreakerStatus) {
	var value float64
	switch status {
	case StatusClosed:
		value = 0
	case StatusOpen:
		value = 1
	case StatusHalfOpen:
		value = 2
	}

	m.circuitBreakerStatus.WithLabelValues(module, subModule).Set(value)
}

// ExportState exports all circuit breaker states as JSON
func (m *Manager) ExportState() ([]byte, error) {
	return m.registry.Export()
}

// ImportState imports circuit breaker states from JSON
func (m *Manager) ImportState(data []byte) error {
	return m.registry.Import(data)
}

// HealthCheck performs a health check on the manager
func (m *Manager) HealthCheck() error {
	// Check if the auto-resume loop is running
	select {
	case <-m.ctx.Done():
		return fmt.Errorf("circuit breaker manager is not running")
	default:
		return nil
	}
}

// parseKey splits a key into module and submodule
func parseKey(key string) (module, subModule string) {
	// Simple parsing: module:submodule or just module
	for i, c := range key {
		if c == ':' {
			return key[:i], key[i+1:]
		}
	}
	return key, ""
}
