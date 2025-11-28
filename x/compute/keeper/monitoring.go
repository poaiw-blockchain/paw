package keeper

import (
	"encoding/json"
	"fmt"
	"time"

	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TASK 95-97: Monitoring hooks, alerting, and telemetry

// MetricsCollector aggregates performance metrics
type MetricsCollector struct {
	TotalJobs              uint64
	CompletedJobs          uint64
	FailedJobs             uint64
	AverageExecutionTime   time.Duration
	TotalProviders         uint64
	ActiveProviders        uint64
	TotalEscrowLocked      math.Int
	SecurityIncidents      uint64
	PanicRecoveries        uint64
	IBCPacketsSent         uint64
	IBCPacketsReceived     uint64
	IBCTimeouts            uint64
}

// MonitoringSeverity defines alert severity levels
type MonitoringSeverity string

const (
	SeverityInfo     MonitoringSeverity = "info"
	SeverityWarning  MonitoringSeverity = "warning"
	SeverityCritical MonitoringSeverity = "critical"
)

// Alert represents a monitoring alert
type Alert struct {
	Timestamp   time.Time
	Severity    MonitoringSeverity
	Category    string
	Message     string
	Source      string
	Metadata    map[string]string
	Acknowledged bool
}

// RecordMetric records a custom metric
func (k Keeper) RecordMetric(ctx sdk.Context, metricName string, value interface{}) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"metric_recorded",
			sdk.NewAttribute("metric", metricName),
			sdk.NewAttribute("value", fmt.Sprintf("%v", value)),
			sdk.NewAttribute("timestamp", ctx.BlockTime().Format(time.RFC3339)),
		),
	)
}

// TrackPerformanceMetric tracks performance-related metrics
func (k Keeper) TrackPerformanceMetric(ctx sdk.Context, operation string, duration time.Duration, success bool) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"performance_metric",
			sdk.NewAttribute("operation", operation),
			sdk.NewAttribute("duration_ms", fmt.Sprintf("%d", duration.Milliseconds())),
			sdk.NewAttribute("success", fmt.Sprintf("%t", success)),
			sdk.NewAttribute("block_height", fmt.Sprintf("%d", ctx.BlockHeight())),
		),
	)

	// Log slow operations
	slowThreshold := 1 * time.Second
	if duration > slowThreshold {
		ctx.Logger().Warn("slow operation detected",
			"operation", operation,
			"duration", duration.String(),
			"threshold", slowThreshold.String(),
		)
	}
}

// MonitorSuspiciousActivity monitors and alerts on suspicious patterns
func (k Keeper) MonitorSuspiciousActivity(
	ctx sdk.Context,
	activityType string,
	source string,
	details string,
	severity MonitoringSeverity,
) error {
	alert := Alert{
		Timestamp: ctx.BlockTime(),
		Severity:  severity,
		Category:  activityType,
		Message:   details,
		Source:    source,
		Metadata:  make(map[string]string),
	}

	// Store alert
	if err := k.storeAlert(ctx, alert); err != nil {
		return err
	}

	// Emit alert event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"security_alert",
			sdk.NewAttribute("type", activityType),
			sdk.NewAttribute("source", source),
			sdk.NewAttribute("severity", string(severity)),
			sdk.NewAttribute("details", details),
			sdk.NewAttribute("timestamp", alert.Timestamp.Format(time.RFC3339)),
		),
	)

	// Log based on severity
	switch severity {
	case SeverityCritical:
		ctx.Logger().Error("CRITICAL SECURITY ALERT",
			"type", activityType,
			"source", source,
			"details", details,
		)
	case SeverityWarning:
		ctx.Logger().Warn("security warning",
			"type", activityType,
			"source", source,
			"details", details,
		)
	default:
		ctx.Logger().Info("security notice",
			"type", activityType,
			"source", source,
		)
	}

	return nil
}

// storeAlert stores an alert in state
func (k Keeper) storeAlert(ctx sdk.Context, alert Alert) error {
	store := ctx.KVStore(k.storeKey)
	key := []byte(fmt.Sprintf("alert_%d_%s", alert.Timestamp.Unix(), alert.Category))

	alertData := map[string]interface{}{
		"severity":  string(alert.Severity),
		"category":  alert.Category,
		"message":   alert.Message,
		"source":    alert.Source,
		"timestamp": alert.Timestamp.Unix(),
	}

	bz, err := json.Marshal(alertData)
	if err != nil {
		return err
	}

	store.Set(key, bz)
	return nil
}

// GetMetrics returns the current compute metrics
func (k Keeper) GetMetrics(ctx sdk.Context) (*MetricsCollector, error) {
	metrics := MetricsCollector{}

	// Count jobs by status
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, []byte("alert_")) // Assuming "alert_" prefix is used for jobs for now
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		metrics.TotalJobs++
		// Additional job status counting would go here
	}

	// Count providers
	providerIterator := storetypes.KVStorePrefixIterator(store, []byte("provider_"))
	defer providerIterator.Close()

	for ; providerIterator.Valid(); providerIterator.Next() {
		metrics.TotalProviders++
		// Check if provider is active
		// metrics.ActiveProviders++
	}
	
	return &metrics, nil
}

// EmitMetricsSnapshot emits a snapshot of current metrics
func (k Keeper) EmitMetricsSnapshot(ctx sdk.Context) {
	metrics, err := k.GetMetrics(ctx)
	if err != nil {
		ctx.Logger().Error("failed to get metrics for snapshot", "error", err)
		return
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"metrics_snapshot",
			sdk.NewAttribute("total_jobs", fmt.Sprintf("%d", metrics.TotalJobs)),
			sdk.NewAttribute("completed_jobs", fmt.Sprintf("%d", metrics.CompletedJobs)),
			sdk.NewAttribute("failed_jobs", fmt.Sprintf("%d", metrics.FailedJobs)),
			sdk.NewAttribute("total_providers", fmt.Sprintf("%d", metrics.TotalProviders)),
			sdk.NewAttribute("active_providers", fmt.Sprintf("%d", metrics.ActiveProviders)),
			sdk.NewAttribute("escrow_locked", metrics.TotalEscrowLocked.String()),
			sdk.NewAttribute("block_height", fmt.Sprintf("%d", ctx.BlockHeight())),
			sdk.NewAttribute("timestamp", ctx.BlockTime().Format(time.RFC3339)),
		),
	)
}

// TrackCircuitBreakerTrigger monitors circuit breaker triggers
func (k Keeper) TrackCircuitBreakerTrigger(ctx sdk.Context, reason string, metadata map[string]string) error {
	return k.MonitorSuspiciousActivity(
		ctx,
		"circuit_breaker_triggered",
		"system",
		reason,
		SeverityCritical,
	)
}

// TrackAnomalousPattern detects and alerts on anomalous patterns
func (k Keeper) TrackAnomalousPattern(
	ctx sdk.Context,
	patternType string,
	source string,
	confidence float64,
) error {
	severity := SeverityInfo
	if confidence > 0.8 {
		severity = SeverityCritical
	} else if confidence > 0.5 {
		severity = SeverityWarning
	}

	details := fmt.Sprintf("Anomalous pattern detected: %s (confidence: %.2f)", patternType, confidence)

	return k.MonitorSuspiciousActivity(ctx, "anomalous_pattern", source, details, severity)
}

// RecordResourceUsage tracks resource consumption
func (k Keeper) RecordResourceUsage(
	ctx sdk.Context,
	resourceType string,
	usage uint64,
	limit uint64,
) {
	utilizationPercent := float64(usage) / float64(limit) * 100

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"resource_usage",
			sdk.NewAttribute("resource_type", resourceType),
			sdk.NewAttribute("usage", fmt.Sprintf("%d", usage)),
			sdk.NewAttribute("limit", fmt.Sprintf("%d", limit)),
			sdk.NewAttribute("utilization", fmt.Sprintf("%.2f%%", utilizationPercent)),
		),
	)

	// Alert if approaching limit
	if utilizationPercent > 80 {
		ctx.Logger().Warn("high resource utilization",
			"resource", resourceType,
			"utilization", fmt.Sprintf("%.2f%%", utilizationPercent),
		)
	}
}

// GetAlerts retrieves recent alerts
func (k Keeper) GetAlerts(ctx sdk.Context, severity MonitoringSeverity, limit int) []Alert {
	store := ctx.KVStore(k.storeKey)
	prefix := []byte("alert_")

	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	alerts := make([]Alert, 0, limit)
	count := 0

	for ; iterator.Valid() && count < limit; iterator.Next() {
		var alertData map[string]interface{}
		if err := json.Unmarshal(iterator.Value(), &alertData); err != nil {
			continue
		}

		alertSeverity := MonitoringSeverity(alertData["severity"].(string))

		// Filter by severity if specified
		if severity != "" && alertSeverity != severity {
			continue
		}

		alert := Alert{
			Severity: alertSeverity,
			Category: alertData["category"].(string),
			Message:  alertData["message"].(string),
			Source:   alertData["source"].(string),
		}

		alerts = append(alerts, alert)
		count++
	}

	return alerts
}
