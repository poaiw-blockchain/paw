package circuit

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// CircuitBreakerStatus represents the state of a circuit breaker
type CircuitBreakerStatus string

const (
	StatusClosed   CircuitBreakerStatus = "closed"    // Normal operation
	StatusOpen     CircuitBreakerStatus = "open"      // Circuit broken, operations blocked
	StatusHalfOpen CircuitBreakerStatus = "half-open" // Testing if system recovered
)

// CircuitBreakerState represents the complete state of a circuit breaker
type CircuitBreakerState struct {
	Module       string                 `json:"module"`
	SubModule    string                 `json:"sub_module,omitempty"` // e.g., specific pool, provider
	Status       CircuitBreakerStatus   `json:"status"`
	PausedAt     *time.Time             `json:"paused_at,omitempty"`
	ResumedAt    *time.Time             `json:"resumed_at,omitempty"`
	PausedBy     string                 `json:"paused_by"` // user ID or system
	Reason       string                 `json:"reason"`
	AutoResumeAt *time.Time             `json:"auto_resume_at,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`

	// History tracking
	TransitionHistory []StateTransition `json:"transition_history"`
}

// StateTransition represents a state change in the circuit breaker
type StateTransition struct {
	From      CircuitBreakerStatus `json:"from"`
	To        CircuitBreakerStatus `json:"to"`
	Timestamp time.Time            `json:"timestamp"`
	Actor     string               `json:"actor"`
	Reason    string               `json:"reason"`
}

// CircuitBreakerRegistry manages all circuit breaker states
type CircuitBreakerRegistry struct {
	mu     sync.RWMutex
	states map[string]*CircuitBreakerState // key: module or module:submodule
}

// NewCircuitBreakerRegistry creates a new registry
func NewCircuitBreakerRegistry() *CircuitBreakerRegistry {
	return &CircuitBreakerRegistry{
		states: make(map[string]*CircuitBreakerState),
	}
}

// Register registers a new circuit breaker
func (r *CircuitBreakerRegistry) Register(module, subModule string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := makeKey(module, subModule)
	if _, exists := r.states[key]; exists {
		return fmt.Errorf("circuit breaker already registered: %s", key)
	}

	r.states[key] = &CircuitBreakerState{
		Module:            module,
		SubModule:         subModule,
		Status:            StatusClosed,
		Metadata:          make(map[string]interface{}),
		TransitionHistory: []StateTransition{},
	}

	return nil
}

// Open opens a circuit breaker (pauses operations)
func (r *CircuitBreakerRegistry) Open(module, subModule, actor, reason string, autoResumeAfter *time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := makeKey(module, subModule)
	state, exists := r.states[key]
	if !exists {
		return fmt.Errorf("circuit breaker not found: %s", key)
	}

	if state.Status == StatusOpen {
		return fmt.Errorf("circuit breaker already open: %s", key)
	}

	now := time.Now()
	oldStatus := state.Status
	state.Status = StatusOpen
	state.PausedAt = &now
	state.PausedBy = actor
	state.Reason = reason
	state.ResumedAt = nil

	if autoResumeAfter != nil {
		resumeTime := now.Add(*autoResumeAfter)
		state.AutoResumeAt = &resumeTime
	}

	// Record transition
	state.TransitionHistory = append(state.TransitionHistory, StateTransition{
		From:      oldStatus,
		To:        StatusOpen,
		Timestamp: now,
		Actor:     actor,
		Reason:    reason,
	})

	return nil
}

// Close closes a circuit breaker (resumes operations)
func (r *CircuitBreakerRegistry) Close(module, subModule, actor, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := makeKey(module, subModule)
	state, exists := r.states[key]
	if !exists {
		return fmt.Errorf("circuit breaker not found: %s", key)
	}

	if state.Status == StatusClosed {
		return fmt.Errorf("circuit breaker already closed: %s", key)
	}

	now := time.Now()
	oldStatus := state.Status
	state.Status = StatusClosed
	state.ResumedAt = &now
	state.AutoResumeAt = nil

	// Record transition
	state.TransitionHistory = append(state.TransitionHistory, StateTransition{
		From:      oldStatus,
		To:        StatusClosed,
		Timestamp: now,
		Actor:     actor,
		Reason:    reason,
	})

	return nil
}

// HalfOpen sets circuit breaker to half-open (testing mode)
func (r *CircuitBreakerRegistry) HalfOpen(module, subModule, actor, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := makeKey(module, subModule)
	state, exists := r.states[key]
	if !exists {
		return fmt.Errorf("circuit breaker not found: %s", key)
	}

	if state.Status != StatusOpen {
		return fmt.Errorf("circuit breaker must be open to transition to half-open: %s", key)
	}

	now := time.Now()
	oldStatus := state.Status
	state.Status = StatusHalfOpen

	// Record transition
	state.TransitionHistory = append(state.TransitionHistory, StateTransition{
		From:      oldStatus,
		To:        StatusHalfOpen,
		Timestamp: now,
		Actor:     actor,
		Reason:    reason,
	})

	return nil
}

// GetState retrieves the current state of a circuit breaker
func (r *CircuitBreakerRegistry) GetState(module, subModule string) (*CircuitBreakerState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := makeKey(module, subModule)
	state, exists := r.states[key]
	if !exists {
		return nil, fmt.Errorf("circuit breaker not found: %s", key)
	}

	// Return a copy to prevent external modifications
	stateCopy := *state
	return &stateCopy, nil
}

// GetAllStates returns all circuit breaker states
func (r *CircuitBreakerRegistry) GetAllStates() map[string]*CircuitBreakerState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*CircuitBreakerState)
	for key, state := range r.states {
		stateCopy := *state
		result[key] = &stateCopy
	}

	return result
}

// IsOpen checks if a circuit breaker is open
func (r *CircuitBreakerRegistry) IsOpen(module, subModule string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := makeKey(module, subModule)
	state, exists := r.states[key]
	if !exists {
		return false // If not registered, allow operations
	}

	return state.Status == StatusOpen
}

// SetMetadata sets metadata for a circuit breaker
func (r *CircuitBreakerRegistry) SetMetadata(module, subModule string, metadata map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := makeKey(module, subModule)
	state, exists := r.states[key]
	if !exists {
		return fmt.Errorf("circuit breaker not found: %s", key)
	}

	for k, v := range metadata {
		state.Metadata[k] = v
	}

	return nil
}

// CheckAutoResume checks if any circuit breakers should auto-resume
func (r *CircuitBreakerRegistry) CheckAutoResume() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	var resumed []string

	for key, state := range r.states {
		if state.Status == StatusOpen && state.AutoResumeAt != nil && now.After(*state.AutoResumeAt) {
			oldStatus := state.Status
			state.Status = StatusClosed
			resumedAt := now
			state.ResumedAt = &resumedAt
			state.AutoResumeAt = nil

			// Record transition
			state.TransitionHistory = append(state.TransitionHistory, StateTransition{
				From:      oldStatus,
				To:        StatusClosed,
				Timestamp: now,
				Actor:     "system",
				Reason:    "auto-resume timer expired",
			})

			resumed = append(resumed, key)
		}
	}

	return resumed
}

// Export exports all states as JSON
func (r *CircuitBreakerRegistry) Export() ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return json.MarshalIndent(r.states, "", "  ")
}

// Import imports states from JSON
func (r *CircuitBreakerRegistry) Import(data []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var states map[string]*CircuitBreakerState
	if err := json.Unmarshal(data, &states); err != nil {
		return fmt.Errorf("failed to unmarshal states: %w", err)
	}

	r.states = states
	return nil
}

// makeKey creates a unique key for a circuit breaker
func makeKey(module, subModule string) string {
	if subModule == "" {
		return module
	}
	return fmt.Sprintf("%s:%s", module, subModule)
}
