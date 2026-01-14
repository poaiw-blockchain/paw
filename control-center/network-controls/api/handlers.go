package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/paw-chain/paw/control-center/network-controls/circuit"
	"github.com/paw-chain/paw/control-center/network-controls/multisig"
)

// Handler provides HTTP handlers for circuit breaker operations
type Handler struct {
	manager          *circuit.Manager
	multiSigVerifier *multisig.Verifier
}

// NewHandler creates a new API handler
func NewHandler(manager *circuit.Manager) *Handler {
	return &Handler{
		manager: manager,
	}
}

// NewHandlerWithMultiSig creates a new API handler with multi-signature verification
func NewHandlerWithMultiSig(manager *circuit.Manager, multiSigConfig *multisig.MultiSigConfig) (*Handler, error) {
	verifier, err := multisig.NewVerifier(multiSigConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create multi-sig verifier: %w", err)
	}
	return &Handler{
		manager:          manager,
		multiSigVerifier: verifier,
	}, nil
}

// RegisterRoutes registers all circuit breaker routes
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// DEX controls
	r.HandleFunc("/api/v1/controls/dex/pause", h.handlePauseDEX).Methods("POST")
	r.HandleFunc("/api/v1/controls/dex/resume", h.handleResumeDEX).Methods("POST")
	r.HandleFunc("/api/v1/controls/dex/pool/{poolID}/pause", h.handlePausePool).Methods("POST")
	r.HandleFunc("/api/v1/controls/dex/pool/{poolID}/resume", h.handleResumePool).Methods("POST")

	// Oracle controls
	r.HandleFunc("/api/v1/controls/oracle/pause", h.handlePauseOracle).Methods("POST")
	r.HandleFunc("/api/v1/controls/oracle/resume", h.handleResumeOracle).Methods("POST")
	r.HandleFunc("/api/v1/controls/oracle/override-price", h.handleOverridePrice).Methods("POST")

	// Compute controls
	r.HandleFunc("/api/v1/controls/compute/pause", h.handlePauseCompute).Methods("POST")
	r.HandleFunc("/api/v1/controls/compute/resume", h.handleResumeCompute).Methods("POST")
	r.HandleFunc("/api/v1/controls/compute/provider/{providerID}/pause", h.handlePauseProvider).Methods("POST")
	r.HandleFunc("/api/v1/controls/compute/provider/{providerID}/resume", h.handleResumeProvider).Methods("POST")
	r.HandleFunc("/api/v1/controls/compute/job/{jobID}/cancel", h.handleCancelJob).Methods("POST")

	// Status and history
	r.HandleFunc("/api/v1/controls/status", h.handleGetStatus).Methods("GET")
	r.HandleFunc("/api/v1/controls/status/{module}", h.handleGetModuleStatus).Methods("GET")
	r.HandleFunc("/api/v1/controls/history", h.handleGetHistory).Methods("GET")

	// Emergency controls
	r.HandleFunc("/api/v1/controls/emergency/halt", h.handleEmergencyHalt).Methods("POST")
	r.HandleFunc("/api/v1/controls/emergency/resume-all", h.handleResumeAll).Methods("POST")

	// Health check
	r.HandleFunc("/api/v1/controls/health", h.handleHealth).Methods("GET")
}

// Request/Response types

type PauseRequest struct {
	Actor           string                 `json:"actor"`
	Reason          string                 `json:"reason"`
	AutoResumeMins  *int                   `json:"auto_resume_mins,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	RequireMultiSig bool                   `json:"require_multi_sig,omitempty"`
}

type ResumeRequest struct {
	Actor  string `json:"actor"`
	Reason string `json:"reason"`
}

type OverridePriceRequest struct {
	Actor    string                 `json:"actor"`
	Pair     string                 `json:"pair"`
	Price    string                 `json:"price"`
	Duration int                    `json:"duration"` // minutes
	Reason   string                 `json:"reason"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type CancelJobRequest struct {
	Actor  string `json:"actor"`
	JobID  string `json:"job_id"`
	Reason string `json:"reason"`
}

type EmergencyHaltRequest struct {
	Actor     string                   `json:"actor"`
	Reason    string                   `json:"reason"`
	Modules   []string                 `json:"modules"`             // empty = all modules
	Signature string                   `json:"signature,omitempty"` // deprecated: use multi_sig instead
	MultiSig  *multisig.MultiSignature `json:"multi_sig,omitempty"` // multi-signature for verification
}

type StatusResponse struct {
	CircuitBreakers map[string]*circuit.CircuitBreakerState `json:"circuit_breakers"`
	Timestamp       time.Time                               `json:"timestamp"`
}

type HistoryResponse struct {
	Module      string                    `json:"module"`
	SubModule   string                    `json:"sub_module,omitempty"`
	Transitions []circuit.StateTransition `json:"transitions"`
}

// DEX Handlers

func (h *Handler) handlePauseDEX(w http.ResponseWriter, r *http.Request) {
	var req PauseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.validatePauseRequest(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var autoResume *time.Duration
	if req.AutoResumeMins != nil {
		d := time.Duration(*req.AutoResumeMins) * time.Minute
		autoResume = &d
	}

	if err := h.manager.PauseModule("dex", "", req.Actor, req.Reason, autoResume); err != nil {
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to pause DEX: %v", err))
		return
	}

	if req.Metadata != nil {
		_ = h.manager.SetMetadata("dex", "", req.Metadata)
	}

	h.writeSuccess(w, map[string]interface{}{
		"status":  "paused",
		"module":  "dex",
		"actor":   req.Actor,
		"reason":  req.Reason,
		"message": "DEX operations paused successfully",
	})
}

func (h *Handler) handleResumeDEX(w http.ResponseWriter, r *http.Request) {
	var req ResumeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.manager.ResumeModule("dex", "", req.Actor, req.Reason); err != nil {
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to resume DEX: %v", err))
		return
	}

	h.writeSuccess(w, map[string]interface{}{
		"status":  "resumed",
		"module":  "dex",
		"actor":   req.Actor,
		"message": "DEX operations resumed successfully",
	})
}

func (h *Handler) handlePausePool(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	poolID := vars["poolID"]

	var req PauseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.validatePauseRequest(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var autoResume *time.Duration
	if req.AutoResumeMins != nil {
		d := time.Duration(*req.AutoResumeMins) * time.Minute
		autoResume = &d
	}

	if err := h.manager.PauseModule("dex", "pool:"+poolID, req.Actor, req.Reason, autoResume); err != nil {
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to pause pool: %v", err))
		return
	}

	h.writeSuccess(w, map[string]interface{}{
		"status":  "paused",
		"module":  "dex",
		"pool_id": poolID,
		"message": fmt.Sprintf("Pool %s paused successfully", poolID),
	})
}

func (h *Handler) handleResumePool(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	poolID := vars["poolID"]

	var req ResumeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.manager.ResumeModule("dex", "pool:"+poolID, req.Actor, req.Reason); err != nil {
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to resume pool: %v", err))
		return
	}

	h.writeSuccess(w, map[string]interface{}{
		"status":  "resumed",
		"module":  "dex",
		"pool_id": poolID,
		"message": fmt.Sprintf("Pool %s resumed successfully", poolID),
	})
}

// Oracle Handlers

func (h *Handler) handlePauseOracle(w http.ResponseWriter, r *http.Request) {
	var req PauseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.validatePauseRequest(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var autoResume *time.Duration
	if req.AutoResumeMins != nil {
		d := time.Duration(*req.AutoResumeMins) * time.Minute
		autoResume = &d
	}

	if err := h.manager.PauseModule("oracle", "", req.Actor, req.Reason, autoResume); err != nil {
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to pause Oracle: %v", err))
		return
	}

	h.writeSuccess(w, map[string]interface{}{
		"status":  "paused",
		"module":  "oracle",
		"message": "Oracle operations paused successfully",
	})
}

func (h *Handler) handleResumeOracle(w http.ResponseWriter, r *http.Request) {
	var req ResumeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.manager.ResumeModule("oracle", "", req.Actor, req.Reason); err != nil {
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to resume Oracle: %v", err))
		return
	}

	h.writeSuccess(w, map[string]interface{}{
		"status":  "resumed",
		"module":  "oracle",
		"message": "Oracle operations resumed successfully",
	})
}

func (h *Handler) handleOverridePrice(w http.ResponseWriter, r *http.Request) {
	var req OverridePriceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Set metadata for the override
	metadata := map[string]interface{}{
		"override_pair":     req.Pair,
		"override_price":    req.Price,
		"override_duration": req.Duration,
		"override_reason":   req.Reason,
		"override_actor":    req.Actor,
	}

	if err := h.manager.SetMetadata("oracle", "price-override", metadata); err != nil {
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to set price override: %v", err))
		return
	}

	h.writeSuccess(w, map[string]interface{}{
		"status":   "override_set",
		"module":   "oracle",
		"pair":     req.Pair,
		"price":    req.Price,
		"duration": req.Duration,
		"message":  fmt.Sprintf("Price override set for %s", req.Pair),
	})
}

// Compute Handlers

func (h *Handler) handlePauseCompute(w http.ResponseWriter, r *http.Request) {
	var req PauseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.validatePauseRequest(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var autoResume *time.Duration
	if req.AutoResumeMins != nil {
		d := time.Duration(*req.AutoResumeMins) * time.Minute
		autoResume = &d
	}

	if err := h.manager.PauseModule("compute", "", req.Actor, req.Reason, autoResume); err != nil {
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to pause Compute: %v", err))
		return
	}

	h.writeSuccess(w, map[string]interface{}{
		"status":  "paused",
		"module":  "compute",
		"message": "Compute operations paused successfully",
	})
}

func (h *Handler) handleResumeCompute(w http.ResponseWriter, r *http.Request) {
	var req ResumeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.manager.ResumeModule("compute", "", req.Actor, req.Reason); err != nil {
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to resume Compute: %v", err))
		return
	}

	h.writeSuccess(w, map[string]interface{}{
		"status":  "resumed",
		"module":  "compute",
		"message": "Compute operations resumed successfully",
	})
}

func (h *Handler) handlePauseProvider(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	providerID := vars["providerID"]

	var req PauseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.validatePauseRequest(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var autoResume *time.Duration
	if req.AutoResumeMins != nil {
		d := time.Duration(*req.AutoResumeMins) * time.Minute
		autoResume = &d
	}

	if err := h.manager.PauseModule("compute", "provider:"+providerID, req.Actor, req.Reason, autoResume); err != nil {
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to pause provider: %v", err))
		return
	}

	h.writeSuccess(w, map[string]interface{}{
		"status":      "paused",
		"module":      "compute",
		"provider_id": providerID,
		"message":     fmt.Sprintf("Provider %s paused successfully", providerID),
	})
}

func (h *Handler) handleResumeProvider(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	providerID := vars["providerID"]

	var req ResumeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.manager.ResumeModule("compute", "provider:"+providerID, req.Actor, req.Reason); err != nil {
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to resume provider: %v", err))
		return
	}

	h.writeSuccess(w, map[string]interface{}{
		"status":      "resumed",
		"module":      "compute",
		"provider_id": providerID,
		"message":     fmt.Sprintf("Provider %s resumed successfully", providerID),
	})
}

func (h *Handler) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["jobID"]

	var req CancelJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Set metadata for job cancellation
	metadata := map[string]interface{}{
		"cancelled_job_id": jobID,
		"cancel_actor":     req.Actor,
		"cancel_reason":    req.Reason,
		"cancel_time":      time.Now(),
	}

	if err := h.manager.SetMetadata("compute", "job-cancel", metadata); err != nil {
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to cancel job: %v", err))
		return
	}

	h.writeSuccess(w, map[string]interface{}{
		"status":  "cancelled",
		"module":  "compute",
		"job_id":  jobID,
		"message": fmt.Sprintf("Job %s cancelled successfully", jobID),
	})
}

// Status and History Handlers

func (h *Handler) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	states := h.manager.GetAllStates()

	response := StatusResponse{
		CircuitBreakers: states,
		Timestamp:       time.Now(),
	}

	h.writeJSON(w, http.StatusOK, response)
}

func (h *Handler) handleGetModuleStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	module := vars["module"]

	state, err := h.manager.GetState(module, "")
	if err != nil {
		h.writeError(w, http.StatusNotFound, fmt.Sprintf("Module not found: %s", module))
		return
	}

	h.writeJSON(w, http.StatusOK, state)
}

func (h *Handler) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	module := r.URL.Query().Get("module")
	subModule := r.URL.Query().Get("sub_module")

	state, err := h.manager.GetState(module, subModule)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "Circuit breaker not found")
		return
	}

	response := HistoryResponse{
		Module:      module,
		SubModule:   subModule,
		Transitions: state.TransitionHistory,
	}

	h.writeJSON(w, http.StatusOK, response)
}

// Emergency Handlers

func (h *Handler) handleEmergencyHalt(w http.ResponseWriter, r *http.Request) {
	var req EmergencyHaltRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Verify multi-signature if verifier is configured
	if h.multiSigVerifier != nil {
		if req.MultiSig == nil {
			h.writeError(w, http.StatusUnauthorized, "Emergency halt requires multi-signature")
			return
		}

		// Verify the multi-signature
		result, err := h.multiSigVerifier.Verify(req.MultiSig)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError,
				fmt.Sprintf("Multi-signature verification error: %v", err))
			return
		}

		if !result.Valid {
			h.writeError(w, http.StatusUnauthorized,
				fmt.Sprintf("Insufficient valid signatures: got %d, need %d. Errors: %v",
					result.ValidSignatures, result.RequiredThreshold, result.Errors))
			return
		}

		// Verify the message content matches the request
		expectedMessage := multisig.CreateSigningMessage("emergency_halt", map[string]interface{}{
			"actor":   req.Actor,
			"reason":  req.Reason,
			"modules": fmt.Sprintf("%v", req.Modules),
		})
		if req.MultiSig.Message != expectedMessage {
			h.writeError(w, http.StatusBadRequest,
				"Signed message does not match request parameters")
			return
		}
	} else {
		// Legacy single signature mode (deprecated)
		if req.Signature == "" && req.MultiSig == nil {
			h.writeError(w, http.StatusUnauthorized, "Emergency halt requires signature")
			return
		}
	}

	modules := req.Modules
	if len(modules) == 0 {
		modules = []string{"dex", "oracle", "compute"}
	}

	for _, module := range modules {
		if err := h.manager.PauseModule(module, "", req.Actor, req.Reason, nil); err != nil {
			h.writeError(w, http.StatusInternalServerError,
				fmt.Sprintf("Failed to halt module %s: %v", module, err))
			return
		}
	}

	h.writeSuccess(w, map[string]interface{}{
		"status":          "halted",
		"modules":         modules,
		"actor":           req.Actor,
		"message":         "Emergency halt executed successfully",
		"multi_sig_valid": req.MultiSig != nil,
	})
}

func (h *Handler) handleResumeAll(w http.ResponseWriter, r *http.Request) {
	var req ResumeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	states := h.manager.GetAllStates()
	resumed := []string{}

	for key, state := range states {
		if state.Status == circuit.StatusOpen {
			module, subModule := parseKeyFromState(key)
			if err := h.manager.ResumeModule(module, subModule, req.Actor, req.Reason); err != nil {
				h.writeError(w, http.StatusInternalServerError,
					fmt.Sprintf("Failed to resume %s: %v", key, err))
				return
			}
			resumed = append(resumed, key)
		}
	}

	h.writeSuccess(w, map[string]interface{}{
		"status":  "resumed",
		"modules": resumed,
		"message": fmt.Sprintf("Resumed %d circuit breakers", len(resumed)),
	})
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := h.manager.HealthCheck(); err != nil {
		h.writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	h.writeSuccess(w, map[string]interface{}{
		"status":  "healthy",
		"message": "Circuit breaker manager is operational",
	})
}

// Helper methods

func (h *Handler) validatePauseRequest(req *PauseRequest) error {
	if req.Actor == "" {
		return fmt.Errorf("actor is required")
	}
	if req.Reason == "" {
		return fmt.Errorf("reason is required")
	}
	return nil
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) writeSuccess(w http.ResponseWriter, data map[string]interface{}) {
	data["success"] = true
	h.writeJSON(w, http.StatusOK, data)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

func parseKeyFromState(key string) (module, subModule string) {
	for i, c := range key {
		if c == ':' {
			return key[:i], key[i+1:]
		}
	}
	return key, ""
}
