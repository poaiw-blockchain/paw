package examples

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	auditlog "paw/control-center/audit-log"
	"paw/control-center/audit-log/middleware"
	"paw/control-center/audit-log/types"

	"github.com/gorilla/mux"
)

// Example: Complete integration with admin API
func IntegrationExample() {
	// 1. Initialize audit log server
	cfg := auditlog.Config{
		DatabaseURL: "postgres://user:pass@localhost/audit_log?sslmode=disable",
		HTTPPort:    8080,
		EnableCORS:  true,
	}

	auditServer, err := auditlog.NewServer(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// 2. Get middleware for automatic logging
	auditLogger := auditServer.GetMiddleware()

	// 3. Create your admin API router
	adminRouter := mux.NewRouter()

	// 4. Apply audit middleware to all routes
	adminRouter.Use(auditLogger.Middleware)
	adminRouter.Use(userContextMiddleware) // Your auth middleware

	// 5. Define admin endpoints
	adminRouter.HandleFunc("/admin/params", updateParamsHandler(auditLogger)).Methods("PUT")
	adminRouter.HandleFunc("/admin/circuit/pause", pauseCircuitHandler(auditLogger)).Methods("POST")
	adminRouter.HandleFunc("/admin/emergency/pause", emergencyPauseHandler(auditLogger)).Methods("POST")
	adminRouter.HandleFunc("/admin/alerts", createAlertHandler(auditLogger)).Methods("POST")

	// 6. Start servers
	go auditServer.Start()

	http.ListenAndServe(":9000", adminRouter)
}

// Example: Update parameters handler with detailed audit logging
func updateParamsHandler(auditLogger *middleware.AuditLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Extract user info from context (set by auth middleware)
		userEmail := ctx.Value("user_email").(string)
		userID := ctx.Value("user_id").(string)
		userRole := ctx.Value("user_role").(string)

		// Parse request
		var params struct {
			Module string                 `json:"module"`
			Params map[string]interface{} `json:"params"`
		}
		// ... decode JSON ...

		// Get current values
		currentParams := getCurrentParams(params.Module)

		// Update parameters
		err := updateParams(params.Module, params.Params)

		// Determine result
		result := types.ResultSuccess
		severity := types.SeverityInfo
		errorMsg := ""
		if err != nil {
			result = types.ResultFailure
			severity = types.SeverityCritical
			errorMsg = err.Error()
		}

		// Log the action with full details
		auditLogger.LogAction(ctx, middleware.AuditAction{
			EventType:     types.EventTypeParamUpdate,
			UserID:        userID,
			UserEmail:     userEmail,
			UserRole:      userRole,
			Action:        fmt.Sprintf("Update %s parameters", params.Module),
			Resource:      params.Module,
			ResourceID:    "params",
			Changes:       params.Params,
			PreviousValue: currentParams,
			NewValue:      params.Params,
			IPAddress:     getClientIP(r),
			UserAgent:     r.UserAgent(),
			SessionID:     ctx.Value("session_id").(string),
			Result:        result,
			ErrorMessage:  errorMsg,
			Severity:      severity,
			Metadata: map[string]interface{}{
				"module":          params.Module,
				"params_count":    len(params.Params),
				"endpoint":        r.URL.Path,
				"request_id":      ctx.Value("request_id"),
			},
		})

		// Send response
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// Example: Circuit breaker pause with audit logging
func pauseCircuitHandler(auditLogger *middleware.AuditLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Parse request
		var req struct {
			Module string `json:"module"`
			Reason string `json:"reason"`
		}
		// ... decode JSON ...

		// Pause circuit
		err := pauseCircuit(req.Module)

		// Log the action
		auditLogger.LogAction(ctx, middleware.AuditAction{
			EventType:    types.EventTypeCircuitPause,
			UserEmail:    ctx.Value("user_email").(string),
			UserRole:     ctx.Value("user_role").(string),
			Action:       fmt.Sprintf("Pause %s circuit breaker", req.Module),
			Resource:     req.Module,
			ResourceID:   "circuit",
			Result:       getResult(err),
			ErrorMessage: getErrorMessage(err),
			Severity:     types.SeverityCritical,
			Metadata: map[string]interface{}{
				"reason": req.Reason,
			},
		})

		// Send response
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// Example: Emergency pause with highest severity
func emergencyPauseHandler(auditLogger *middleware.AuditLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Parse request
		var req struct {
			Reason   string `json:"reason"`
			Duration int    `json:"duration_seconds"`
		}
		// ... decode JSON ...

		// Execute emergency pause
		err := executeEmergencyPause(req.Duration)

		// Log with critical severity
		auditLogger.LogAction(ctx, middleware.AuditAction{
			EventType:    types.EventTypeEmergencyPause,
			UserEmail:    ctx.Value("user_email").(string),
			UserRole:     ctx.Value("user_role").(string),
			Action:       "Execute emergency pause",
			Resource:     "system",
			ResourceID:   "emergency",
			Result:       getResult(err),
			ErrorMessage: getErrorMessage(err),
			Severity:     types.SeverityCritical,
			Metadata: map[string]interface{}{
				"reason":           req.Reason,
				"duration_seconds": req.Duration,
				"triggered_by":     ctx.Value("user_email"),
			},
		})

		// Send response
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// Example: Alert creation with audit logging
func createAlertHandler(auditLogger *middleware.AuditLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Parse request
		var alert struct {
			Name      string `json:"name"`
			Condition string `json:"condition"`
			Threshold float64 `json:"threshold"`
		}
		// ... decode JSON ...

		// Create alert
		alertID, err := createAlert(alert.Name, alert.Condition, alert.Threshold)

		// Log the action
		auditLogger.LogAction(ctx, middleware.AuditAction{
			EventType:  types.EventTypeAlertRuleCreated,
			UserEmail:  ctx.Value("user_email").(string),
			UserRole:   ctx.Value("user_role").(string),
			Action:     fmt.Sprintf("Create alert rule: %s", alert.Name),
			Resource:   "alert",
			ResourceID: alertID,
			NewValue:   alert,
			Result:     getResult(err),
			ErrorMessage: getErrorMessage(err),
			Severity:   types.SeverityInfo,
			Metadata: map[string]interface{}{
				"alert_name": alert.Name,
				"condition":  alert.Condition,
				"threshold":  alert.Threshold,
			},
		})

		// Send response
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}
}

// Example: Query audit logs programmatically
func QueryAuditLogsExample(auditServer *auditlog.Server) {
	ctx := context.Background()
	storage := auditServer.GetStorage()

	// Query logs for a specific user
	filters := types.QueryFilters{
		UserEmail: "admin@example.com",
		EventType: []types.EventType{
			types.EventTypeParamUpdate,
			types.EventTypeCircuitPause,
		},
		StartTime: timeFromDaysAgo(7),
		Limit:     100,
	}

	entries, total, err := storage.Query(ctx, filters)
	if err != nil {
		log.Printf("Failed to query logs: %v", err)
		return
	}

	log.Printf("Found %d entries (total: %d)", len(entries), total)
	for _, entry := range entries {
		log.Printf("[%s] %s: %s - %s",
			entry.Timestamp.Format("2006-01-02 15:04:05"),
			entry.UserEmail,
			entry.Action,
			entry.Result,
		)
	}
}

// Helper functions
func userContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract user from JWT or session
		// Set in context
		ctx := r.Context()
		ctx = context.WithValue(ctx, "user_email", "admin@example.com")
		ctx = context.WithValue(ctx, "user_id", "user123")
		ctx = context.WithValue(ctx, "user_role", "admin")
		ctx = context.WithValue(ctx, "session_id", "session123")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getClientIP(r *http.Request) string {
	// Implementation
	return r.RemoteAddr
}

func getResult(err error) types.Result {
	if err != nil {
		return types.ResultFailure
	}
	return types.ResultSuccess
}

func getErrorMessage(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

// Stub functions for example
func getCurrentParams(module string) map[string]interface{} { return nil }
func updateParams(module string, params map[string]interface{}) error { return nil }
func pauseCircuit(module string) error { return nil }
func executeEmergencyPause(duration int) error { return nil }
func createAlert(name, condition string, threshold float64) (string, error) { return "alert123", nil }
func timeFromDaysAgo(days int) time.Time { return time.Now().AddDate(0, 0, -days) }
