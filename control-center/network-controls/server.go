package networkcontrols

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/paw-chain/paw/control-center/network-controls/api"
	"github.com/paw-chain/paw/control-center/network-controls/circuit"
	"github.com/paw-chain/paw/control-center/network-controls/integration"
	computekeeper "github.com/paw-chain/paw/x/compute/keeper"
	dexkeeper "github.com/paw-chain/paw/x/dex/keeper"
	oraclekeeper "github.com/paw-chain/paw/x/oracle/keeper"
)

// Server is the network controls server
type Server struct {
	manager *circuit.Manager
	handler *api.Handler

	// Module integrations
	dexIntegration     *integration.DEXIntegration
	oracleIntegration  *integration.OracleIntegration
	computeIntegration *integration.ComputeIntegration

	// HTTP server
	httpServer *http.Server
	router     *mux.Router

	// Context provider for SDK operations
	ctxProvider func() sdk.Context

	// Shutdown
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Config holds server configuration
type Config struct {
	ListenAddr string
	EnableCORS bool
}

// NewServer creates a new network controls server
func NewServer(
	cfg Config,
	dexKeeper *dexkeeper.Keeper,
	oracleKeeper *oraclekeeper.Keeper,
	computeKeeper *computekeeper.Keeper,
	ctxProvider func() sdk.Context,
) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	manager := circuit.NewManager()
	handler := api.NewHandler(manager)

	router := mux.NewRouter()

	s := &Server{
		manager:            manager,
		handler:            handler,
		dexIntegration:     integration.NewDEXIntegration(dexKeeper),
		oracleIntegration:  integration.NewOracleIntegration(oracleKeeper),
		computeIntegration: integration.NewComputeIntegration(computeKeeper),
		router:             router,
		ctxProvider:        ctxProvider,
		ctx:                ctx,
		cancel:             cancel,
	}

	// Register routes
	handler.RegisterRoutes(router)

	// Apply middleware
	var httpHandler http.Handler = router
	if cfg.EnableCORS {
		httpHandler = handlers.CORS(
			handlers.AllowedOrigins([]string{"*"}),
			handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
			handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
		)(router)
	}

	s.httpServer = &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      httpHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Set up auto-resume callback
	manager.SetAutoResumeCallback(s.handleAutoResume)

	return s
}

// Start starts the network controls server
func (s *Server) Start() error {
	// Register circuit breakers
	if err := s.registerCircuitBreakers(); err != nil {
		return fmt.Errorf("failed to register circuit breakers: %w", err)
	}

	// Start circuit breaker manager
	s.manager.Start()

	// Start HTTP server
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		log.Printf("Network controls server listening on %s", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Start sync loop
	s.wg.Add(1)
	go s.syncLoop()

	return nil
}

// Stop stops the network controls server
func (s *Server) Stop() error {
	s.cancel()

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Stop circuit breaker manager
	s.manager.Stop()

	s.wg.Wait()
	log.Println("Network controls server stopped")
	return nil
}

// registerCircuitBreakers registers all module circuit breakers
func (s *Server) registerCircuitBreakers() error {
	modules := []struct {
		name       string
		submodules []string
	}{
		{"dex", []string{""}},
		{"oracle", []string{""}},
		{"compute", []string{""}},
	}

	for _, mod := range modules {
		for _, sub := range mod.submodules {
			if err := s.manager.RegisterCircuitBreaker(mod.name, sub); err != nil {
				return err
			}
		}
	}

	return nil
}

// syncLoop syncs circuit breaker state between control center and SDK modules
func (s *Server) syncLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.syncState()
		}
	}
}

// syncState syncs circuit breaker state to SDK modules
func (s *Server) syncState() {
	if s.ctxProvider == nil {
		return
	}

	sdkCtx := s.ctxProvider()

	// Sync DEX
	dexState, err := s.manager.GetState("dex", "")
	if err == nil {
		dexEnabled, _, _ := s.dexIntegration.GetState(sdkCtx)
		if dexState.Status == circuit.StatusOpen && !dexEnabled {
			// Manager says open, but SDK says closed - sync to SDK
			_ = s.dexIntegration.Pause(sdkCtx, dexState.PausedBy, dexState.Reason)
		} else if dexState.Status == circuit.StatusClosed && dexEnabled {
			// Manager says closed, but SDK says open - sync to SDK
			_ = s.dexIntegration.Resume(sdkCtx, "system", "auto-sync")
		}
	}

	// Sync Oracle
	oracleState, err := s.manager.GetState("oracle", "")
	if err == nil {
		oracleEnabled, _, _ := s.oracleIntegration.GetState(sdkCtx)
		if oracleState.Status == circuit.StatusOpen && !oracleEnabled {
			_ = s.oracleIntegration.Pause(sdkCtx, oracleState.PausedBy, oracleState.Reason)
		} else if oracleState.Status == circuit.StatusClosed && oracleEnabled {
			_ = s.oracleIntegration.Resume(sdkCtx, "system", "auto-sync")
		}
	}

	// Sync Compute
	computeState, err := s.manager.GetState("compute", "")
	if err == nil {
		computeEnabled, _, _ := s.computeIntegration.GetState(sdkCtx)
		if computeState.Status == circuit.StatusOpen && !computeEnabled {
			_ = s.computeIntegration.Pause(sdkCtx, computeState.PausedBy, computeState.Reason)
		} else if computeState.Status == circuit.StatusClosed && computeEnabled {
			_ = s.computeIntegration.Resume(sdkCtx, "system", "auto-sync")
		}
	}
}

// handleAutoResume is called when a circuit breaker auto-resumes
func (s *Server) handleAutoResume(module, subModule string) error {
	if s.ctxProvider == nil {
		return nil
	}

	sdkCtx := s.ctxProvider()

	switch module {
	case "dex":
		if subModule == "" {
			return s.dexIntegration.Resume(sdkCtx, "system", "auto-resume")
		}
		// Handle pool-specific auto-resume if needed
	case "oracle":
		if subModule == "" {
			return s.oracleIntegration.Resume(sdkCtx, "system", "auto-resume")
		}
		// Handle feed-specific auto-resume if needed
	case "compute":
		if subModule == "" {
			return s.computeIntegration.Resume(sdkCtx, "system", "auto-resume")
		}
		// Handle provider-specific auto-resume if needed
	}

	return nil
}

// GetManager returns the circuit breaker manager
func (s *Server) GetManager() *circuit.Manager {
	return s.manager
}

// GetDEXIntegration returns the DEX integration
func (s *Server) GetDEXIntegration() *integration.DEXIntegration {
	return s.dexIntegration
}

// GetOracleIntegration returns the Oracle integration
func (s *Server) GetOracleIntegration() *integration.OracleIntegration {
	return s.oracleIntegration
}

// GetComputeIntegration returns the Compute integration
func (s *Server) GetComputeIntegration() *integration.ComputeIntegration {
	return s.computeIntegration
}

// HealthCheck performs a health check on the server
func (s *Server) HealthCheck() error {
	if err := s.manager.HealthCheck(); err != nil {
		return err
	}

	// Additional health checks could go here

	return nil
}
