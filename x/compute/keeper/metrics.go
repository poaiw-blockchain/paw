package keeper

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ComputeMetrics holds all Prometheus metrics for the Compute module
type ComputeMetrics struct {
	// Job metrics
	JobsSubmitted    *prometheus.CounterVec
	JobsAccepted     *prometheus.CounterVec
	JobsCompleted    *prometheus.CounterVec
	JobsFailed       *prometheus.CounterVec
	JobExecutionTime prometheus.Histogram
	JobQueueSize     prometheus.Gauge

	// ZK proof metrics
	ProofsVerified         *prometheus.CounterVec
	ProofVerificationTime  prometheus.Histogram
	InvalidProofs          *prometheus.CounterVec
	CircuitInitializations prometheus.Counter

	// Escrow metrics
	EscrowLocked   *prometheus.CounterVec
	EscrowReleased *prometheus.CounterVec
	EscrowRefunded *prometheus.CounterVec
	EscrowBalance  *prometheus.GaugeVec

	// Provider metrics
	ProvidersRegistered *prometheus.CounterVec
	ProvidersActive     prometheus.Gauge
	ProviderReputation  *prometheus.GaugeVec
	ProviderStake       *prometheus.GaugeVec
	ProviderSlashing    *prometheus.CounterVec

	// IBC compute metrics
	IBCJobsDistributed   *prometheus.CounterVec
	IBCResultsReceived   *prometheus.CounterVec
	RemoteProvidersCount *prometheus.GaugeVec
	CrossChainLatency    *prometheus.HistogramVec
	IBCTimeouts          *prometheus.CounterVec

	// Security metrics
	SecurityIncidents      *prometheus.CounterVec
	PanicRecoveries        prometheus.Counter
	RateLimitExceeds       *prometheus.CounterVec
	CircuitBreakerTriggers *prometheus.CounterVec

	// Performance metrics
	CircuitCompilations prometheus.Counter
	StateRecoveries     prometheus.Counter
	TimeoutCleanups     prometheus.Counter
	StaleJobCleanups    prometheus.Counter
	NonceCleanups       prometheus.Counter
	NoncesCleanedTotal  prometheus.Counter
}

var (
	computeMetricsOnce sync.Once
	computeMetrics     *ComputeMetrics
)

// NewComputeMetrics creates and registers Compute metrics (singleton pattern)
func NewComputeMetrics() *ComputeMetrics {
	computeMetricsOnce.Do(func() {
		computeMetrics = &ComputeMetrics{
			// Job metrics
			JobsSubmitted: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "jobs_submitted_total",
					Help:      "Total compute jobs submitted",
				},
				[]string{"job_type"},
			),
			JobsAccepted: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "jobs_accepted_total",
					Help:      "Total jobs accepted by providers",
				},
				[]string{"provider"},
			),
			JobsCompleted: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "jobs_completed_total",
					Help:      "Total jobs completed",
				},
				[]string{"provider", "status"},
			),
			JobsFailed: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "jobs_failed_total",
					Help:      "Total jobs failed",
				},
				[]string{"provider", "reason"},
			),
			JobExecutionTime: promauto.NewHistogram(
				prometheus.HistogramOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "job_execution_seconds",
					Help:      "Job execution time in seconds",
					Buckets:   []float64{1, 5, 10, 30, 60, 300, 600},
				},
			),
			JobQueueSize: promauto.NewGauge(
				prometheus.GaugeOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "job_queue_size",
					Help:      "Current number of jobs in queue",
				},
			),

			// ZK proof metrics
			ProofsVerified: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "proofs_verified_total",
					Help:      "Total ZK proofs verified",
				},
				[]string{"proof_type", "status"},
			),
			ProofVerificationTime: promauto.NewHistogram(
				prometheus.HistogramOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "proof_verification_seconds",
					Help:      "ZK proof verification time",
					Buckets:   []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0},
				},
			),
			InvalidProofs: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "invalid_proofs_total",
					Help:      "Invalid ZK proofs submitted",
				},
				[]string{"provider", "reason"},
			),
			CircuitInitializations: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "circuit_initializations_total",
					Help:      "ZK circuit initialization events",
				},
			),

			// Escrow metrics
			EscrowLocked: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "escrow_locked_total",
					Help:      "Total escrow locked",
				},
				[]string{"denom"},
			),
			EscrowReleased: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "escrow_released_total",
					Help:      "Total escrow released to providers",
				},
				[]string{"denom"},
			),
			EscrowRefunded: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "escrow_refunded_total",
					Help:      "Total escrow refunded to requesters",
				},
				[]string{"denom"},
			),
			EscrowBalance: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "escrow_balance",
					Help:      "Current escrow balance",
				},
				[]string{"denom"},
			),

			// Provider metrics
			ProvidersRegistered: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "providers_registered_total",
					Help:      "Total providers registered",
				},
				[]string{"capability"},
			),
			ProvidersActive: promauto.NewGauge(
				prometheus.GaugeOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "providers_active",
					Help:      "Currently active providers",
				},
			),
			ProviderReputation: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "provider_reputation_score",
					Help:      "Provider reputation score (0-100)",
				},
				[]string{"provider"},
			),
			ProviderStake: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "provider_stake",
					Help:      "Provider stake amount",
				},
				[]string{"provider", "denom"},
			),
			ProviderSlashing: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "provider_slashing_events_total",
					Help:      "Provider slashing events",
				},
				[]string{"provider", "reason"},
			),

			// IBC compute metrics
			IBCJobsDistributed: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "ibc_jobs_distributed_total",
					Help:      "Jobs distributed to remote chains",
				},
				[]string{"target_chain"},
			),
			IBCResultsReceived: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "ibc_results_received_total",
					Help:      "Results received from remote chains",
				},
				[]string{"source_chain"},
			),
			RemoteProvidersCount: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "remote_providers_discovered",
					Help:      "Remote providers discovered per chain",
				},
				[]string{"chain"},
			),
			CrossChainLatency: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "cross_chain_latency_seconds",
					Help:      "Cross-chain job execution latency",
					Buckets:   []float64{5, 10, 30, 60, 120, 300},
				},
				[]string{"chain"},
			),
			IBCTimeouts: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "ibc_timeouts_total",
					Help:      "IBC compute packet timeouts",
				},
				[]string{"chain"},
			),

			// Security metrics
			SecurityIncidents: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "security_incidents_total",
					Help:      "Security incidents detected",
				},
				[]string{"type", "severity"},
			),
			PanicRecoveries: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "panic_recoveries_total",
					Help:      "Panic recovery events",
				},
			),
			RateLimitExceeds: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "rate_limits_exceeded_total",
					Help:      "Rate limit violations",
				},
				[]string{"operation", "user"},
			),
			CircuitBreakerTriggers: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "circuit_breaker_triggers_total",
					Help:      "Circuit breaker activations",
				},
				[]string{"reason"},
			),

			// Performance metrics
			CircuitCompilations: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "circuit_compilations_total",
					Help:      "ZK circuit compilation events",
				},
			),
			StateRecoveries: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "state_recoveries_total",
					Help:      "State recovery operations",
				},
			),
			TimeoutCleanups: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "timeout_cleanups_total",
					Help:      "Timeout cleanup operations",
				},
			),
			StaleJobCleanups: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "stale_job_cleanups_total",
					Help:      "Stale job cleanup operations",
				},
			),
			NonceCleanups: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "nonce_cleanups_total",
					Help:      "Nonce cleanup operations executed",
				},
			),
			NoncesCleanedTotal: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "compute",
					Name:      "nonces_cleaned_total",
					Help:      "Total number of nonces cleaned up",
				},
			),
		}
	})
	return computeMetrics
}

// GetComputeMetrics returns the singleton Compute metrics instance
func GetComputeMetrics() *ComputeMetrics {
	if computeMetrics == nil {
		return NewComputeMetrics()
	}
	return computeMetrics
}

// RecordNonceCleanup records nonce cleanup metrics
func (m *ComputeMetrics) RecordNonceCleanup(cleanedCount int) {
	if m == nil {
		return
	}
	m.NonceCleanups.Inc()
	m.NoncesCleanedTotal.Add(float64(cleanedCount))
}
