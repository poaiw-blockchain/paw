package reputation

import (
	"math"
	"time"
)

// Scorer handles peer reputation scoring
type Scorer struct {
	weights ScoreWeights
	config  ScoringConfig
}

// ScoringConfig contains scoring configuration
type ScoringConfig struct {
	// Time windows
	ScoreDecayPeriod time.Duration // How often to decay scores
	ScoreDecayFactor float64       // Decay multiplier (0.0-1.0)
	MinScoreHistory  time.Duration // Min time to keep historical data

	// Thresholds
	MinUptimeForGoodScore  time.Duration // Minimum uptime for positive scoring
	MaxLatencyForGoodScore time.Duration // Max latency considered good
	MinValidMessageRatio   float64       // Min ratio for good score
	FastBlockThreshold     time.Duration // Threshold for "fast" block propagation

	// Penalties
	ViolationPenalty        float64 // Score reduction per violation
	DoubleSignPenalty       float64 // Severe penalty for double signing
	InvalidBlockPenalty     float64 // Penalty for invalid blocks
	SpamPenalty             float64 // Penalty for spam
	MalformedMessagePenalty float64 // Penalty for malformed messages

	// Scoring caps
	MaxScore          float64 // Maximum achievable score
	MinScore          float64 // Minimum score (before ban)
	NewPeerStartScore float64 // Starting score for new peers
}

// DefaultScoringConfig returns default scoring configuration
func DefaultScoringConfig() ScoringConfig {
	return ScoringConfig{
		ScoreDecayPeriod: 24 * time.Hour,
		ScoreDecayFactor: 0.95, // 5% decay per day
		MinScoreHistory:  7 * 24 * time.Hour,

		MinUptimeForGoodScore:  1 * time.Hour,
		MaxLatencyForGoodScore: 500 * time.Millisecond,
		MinValidMessageRatio:   0.95, // 95% valid messages
		FastBlockThreshold:     1 * time.Second,

		ViolationPenalty:        5.0,
		DoubleSignPenalty:       100.0, // Instant ban
		InvalidBlockPenalty:     20.0,
		SpamPenalty:             10.0,
		MalformedMessagePenalty: 2.0,

		MaxScore:          100.0,
		MinScore:          0.0,
		NewPeerStartScore: 50.0, // Neutral starting score
	}
}

// NewScorer creates a new peer scorer
func NewScorer(weights ScoreWeights, config *ScoringConfig) *Scorer {
	var cfg ScoringConfig
	if config != nil {
		cfg = *config
	}
	return &Scorer{
		weights: weights,
		config:  cfg,
	}
}

// CalculateScore computes the composite reputation score for a peer
func (s *Scorer) CalculateScore(rep *PeerReputation) float64 {
	// Start with base score
	score := 0.0

	// 1. Uptime score (0-100)
	uptimeScore := s.calculateUptimeScore(rep)
	score += uptimeScore * s.weights.UptimeWeight

	// 2. Message validity score (0-100)
	validityScore := s.calculateValidityScore(rep)
	score += validityScore * s.weights.MessageValidityWeight

	// 3. Latency score (0-100)
	latencyScore := s.calculateLatencyScore(rep)
	score += latencyScore * s.weights.LatencyWeight

	// 4. Block propagation score (0-100)
	blockScore := s.calculateBlockPropagationScore(rep)
	score += blockScore * s.weights.BlockPropWeight

	// 5. Apply violation penalties
	violationPenalty := s.calculateViolationPenalty(rep)
	score -= violationPenalty

	// Apply age decay for inactive peers
	ageDecay := s.calculateAgeDecay(rep)
	score *= ageDecay

	// Clamp to valid range
	score = math.Max(s.config.MinScore, math.Min(s.config.MaxScore, score))

	return score
}

// calculateUptimeScore scores based on uptime and reliability
func (s *Scorer) calculateUptimeScore(rep *PeerReputation) float64 {
	if rep.Metrics.ConnectionCount == 0 {
		return s.config.NewPeerStartScore
	}

	// Calculate uptime ratio
	totalTime := time.Since(rep.FirstSeen)
	if totalTime == 0 {
		return s.config.NewPeerStartScore
	}

	uptimeRatio := float64(rep.Metrics.TotalUptime) / float64(totalTime)
	uptimeRatio = math.Min(1.0, uptimeRatio)

	// Calculate connection stability
	// Penalize frequent disconnections
	avgSessionTime := float64(rep.Metrics.TotalUptime) / float64(rep.Metrics.ConnectionCount)
	stabilityScore := 1.0
	if rep.Metrics.DisconnectionCount > 0 {
		disconnectRatio := float64(rep.Metrics.DisconnectionCount) / float64(rep.Metrics.ConnectionCount)
		stabilityScore = math.Max(0.0, 1.0-disconnectRatio*0.5)
	}

	// Combine uptime and stability
	baseScore := (uptimeRatio*0.7 + stabilityScore*0.3) * 100

	// Bonus for long-term stable peers
	if rep.Metrics.TotalUptime >= s.config.MinUptimeForGoodScore {
		if avgSessionTime >= float64(s.config.MinUptimeForGoodScore) {
			baseScore *= 1.1 // 10% bonus
		}
	}

	return math.Min(100.0, baseScore)
}

// calculateValidityScore scores message validity ratio
func (s *Scorer) calculateValidityScore(rep *PeerReputation) float64 {
	if rep.Metrics.TotalMessages == 0 {
		return s.config.NewPeerStartScore
	}

	// Calculate cached ratio if not already done
	if rep.Metrics.ValidMessageRatio == 0 && rep.Metrics.TotalMessages > 0 {
		rep.Metrics.ValidMessageRatio = float64(rep.Metrics.ValidMessages) / float64(rep.Metrics.TotalMessages)
	}

	ratio := rep.Metrics.ValidMessageRatio

	// Non-linear scoring - heavily penalize below threshold
	if ratio >= s.config.MinValidMessageRatio {
		// Linear from 95% to 100%
		return 80.0 + (ratio-s.config.MinValidMessageRatio)/(1.0-s.config.MinValidMessageRatio)*20.0
	} else if ratio >= 0.80 {
		// Moderate scores for 80-95%
		return 40.0 + (ratio-0.80)/(s.config.MinValidMessageRatio-0.80)*40.0
	} else {
		// Poor scores below 80%
		return ratio * 50.0
	}
}

// calculateLatencyScore scores based on response latency
func (s *Scorer) calculateLatencyScore(rep *PeerReputation) float64 {
	if rep.Metrics.LatencyMeasurements == 0 {
		return s.config.NewPeerStartScore
	}

	avgLatency := rep.Metrics.AvgResponseLatency
	maxGoodLatency := s.config.MaxLatencyForGoodScore

	// Exponential decay scoring
	// 0-500ms: 80-100 points
	// 500-2000ms: 40-80 points
	// 2000ms+: 0-40 points

	if avgLatency <= maxGoodLatency {
		// Excellent latency
		ratio := 1.0 - (float64(avgLatency) / float64(maxGoodLatency))
		return 80.0 + ratio*20.0
	} else if avgLatency <= 2*time.Second {
		// Moderate latency
		ratio := 1.0 - (float64(avgLatency-maxGoodLatency) / float64(2*time.Second-maxGoodLatency))
		return 40.0 + ratio*40.0
	} else {
		// Poor latency - exponential penalty
		excessMs := float64(avgLatency-2*time.Second) / float64(time.Millisecond)
		penalty := math.Min(40.0, excessMs/100.0)
		return math.Max(0.0, 40.0-penalty)
	}
}

// calculateBlockPropagationScore scores block propagation speed
func (s *Scorer) calculateBlockPropagationScore(rep *PeerReputation) float64 {
	if rep.Metrics.BlocksPropagated == 0 {
		return s.config.NewPeerStartScore
	}

	// Calculate fast block ratio
	fastRatio := float64(rep.Metrics.FastBlockCount) / float64(rep.Metrics.BlocksPropagated)
	avgPropTime := rep.Metrics.AvgBlockPropagation

	// Score based on average propagation time
	fastThreshold := s.config.FastBlockThreshold
	var baseScore float64

	if avgPropTime <= fastThreshold {
		// Excellent propagation
		baseScore = 90.0 + (1.0-(float64(avgPropTime)/float64(fastThreshold)))*10.0
	} else if avgPropTime <= 5*time.Second {
		// Good propagation
		ratio := 1.0 - (float64(avgPropTime-fastThreshold) / float64(5*time.Second-fastThreshold))
		baseScore = 60.0 + ratio*30.0
	} else if avgPropTime <= 30*time.Second {
		// Moderate propagation
		ratio := 1.0 - (float64(avgPropTime-5*time.Second) / float64(30*time.Second-5*time.Second))
		baseScore = 30.0 + ratio*30.0
	} else {
		// Slow propagation
		baseScore = math.Max(0.0, 30.0-(float64(avgPropTime-30*time.Second)/float64(time.Second)))
	}

	// Bonus for high fast block ratio
	if fastRatio >= 0.8 {
		baseScore *= 1.1
	}

	return math.Min(100.0, baseScore)
}

// calculateViolationPenalty calculates penalty from protocol violations
func (s *Scorer) calculateViolationPenalty(rep *PeerReputation) float64 {
	penalty := 0.0

	// Standard violations
	penalty += float64(rep.Metrics.ProtocolViolations) * s.config.ViolationPenalty

	// Severe violations
	penalty += float64(rep.Metrics.DoubleSignAttempts) * s.config.DoubleSignPenalty
	penalty += float64(rep.Metrics.InvalidBlockProps) * s.config.InvalidBlockPenalty
	penalty += float64(rep.Metrics.SpamAttempts) * s.config.SpamPenalty
	penalty += float64(rep.Metrics.MalformedMessages) * s.config.MalformedMessagePenalty

	// DoS and security violations
	penalty += float64(rep.Metrics.OversizedMessages) * 15.0   // Severe DoS penalty
	penalty += float64(rep.Metrics.BandwidthViolations) * 10.0 // Bandwidth abuse
	penalty += float64(rep.Metrics.SecurityEvents) * 20.0      // Critical security events

	// Streak multiplier - repeated violations within short time are worse
	if rep.Metrics.ViolationStreak > 3 {
		streakMultiplier := 1.0 + (float64(rep.Metrics.ViolationStreak-3) * 0.2) // +20% per violation over 3
		penalty *= math.Min(streakMultiplier, 3.0)                               // Cap at 3x multiplier
	}

	// Total penalty points contribute to overall score reduction
	penalty += float64(rep.Metrics.TotalPenaltyPoints) * 0.5

	// Weight the penalty
	penalty *= s.weights.ViolationPenalty

	return penalty
}

// calculateAgeDecay applies decay for inactive peers
func (s *Scorer) calculateAgeDecay(rep *PeerReputation) float64 {
	timeSinceLastSeen := time.Since(rep.LastSeen)

	// No decay if seen recently (within decay period)
	if timeSinceLastSeen < s.config.ScoreDecayPeriod {
		return 1.0
	}

	// Apply exponential decay
	periods := float64(timeSinceLastSeen) / float64(s.config.ScoreDecayPeriod)
	decay := math.Pow(s.config.ScoreDecayFactor, periods)

	return math.Max(0.1, decay) // Minimum 10% of score retained
}

// ApplyEvent updates metrics based on an event
func (s *Scorer) ApplyEvent(rep *PeerReputation, event *PeerEvent) {
	if event == nil {
		return
	}
	now := event.Timestamp
	if now.IsZero() {
		now = time.Now()
	}

	switch event.EventType {
	case EventTypeConnected:
		rep.Metrics.ConnectionCount++
		rep.Metrics.LastUptimeUpdate = now
		rep.LastSeen = now
		if rep.FirstSeen.IsZero() {
			rep.FirstSeen = now
		}

	case EventTypeDisconnected:
		rep.Metrics.DisconnectionCount++
		// Update uptime
		if !rep.Metrics.LastUptimeUpdate.IsZero() {
			sessionTime := now.Sub(rep.Metrics.LastUptimeUpdate)
			rep.Metrics.TotalUptime += sessionTime
		}
		rep.LastSeen = now

	case EventTypeValidMessage:
		rep.Metrics.ValidMessages++
		rep.Metrics.TotalMessages++
		rep.Metrics.ValidMessageRatio = float64(rep.Metrics.ValidMessages) / float64(rep.Metrics.TotalMessages)
		rep.LastSeen = now

		// Track bytes
		if event.Data.MessageSize > 0 {
			rep.Metrics.BytesReceived += event.Data.MessageSize
		}

	case EventTypeInvalidMessage:
		rep.Metrics.InvalidMessages++
		rep.Metrics.TotalMessages++
		rep.Metrics.ValidMessageRatio = float64(rep.Metrics.ValidMessages) / float64(rep.Metrics.TotalMessages)
		rep.Metrics.MalformedMessages++
		rep.LastSeen = now

	case EventTypeBlockPropagated:
		rep.Metrics.BlocksPropagated++
		if event.Data.Latency > 0 {
			// Update average propagation time
			totalTime := rep.Metrics.AvgBlockPropagation * time.Duration(rep.Metrics.BlocksPropagated-1)
			rep.Metrics.AvgBlockPropagation = (totalTime + event.Data.Latency) / time.Duration(rep.Metrics.BlocksPropagated)

			// Track fast blocks
			if event.Data.Latency <= s.config.FastBlockThreshold {
				rep.Metrics.FastBlockCount++
			}
		}
		rep.LastSeen = now

	case EventTypeProtocolViolation:
		rep.Metrics.ProtocolViolations++
		rep.LastSeen = now

	case EventTypeDoubleSign:
		rep.Metrics.DoubleSignAttempts++
		rep.Metrics.ProtocolViolations++
		rep.LastSeen = now

	case EventTypeInvalidBlock:
		rep.Metrics.InvalidBlockProps++
		rep.Metrics.ProtocolViolations++
		rep.LastSeen = now

	case EventTypeSpam:
		rep.Metrics.SpamAttempts++
		rep.Metrics.ProtocolViolations++
		rep.Metrics.LastViolation = now
		rep.Metrics.ViolationStreak++
		rep.Metrics.TotalPenaltyPoints += 10
		rep.LastSeen = now

	case EventTypeOversizedMessage:
		rep.Metrics.OversizedMessages++
		rep.Metrics.ProtocolViolations++
		rep.Metrics.LastViolation = now
		rep.Metrics.ViolationStreak++
		rep.Metrics.TotalPenaltyPoints += 15 // Severe penalty for DoS attempts
		rep.LastSeen = now

	case EventTypeBandwidthAbuse:
		rep.Metrics.BandwidthViolations++
		rep.Metrics.ProtocolViolations++
		rep.Metrics.LastViolation = now
		rep.Metrics.ViolationStreak++
		rep.Metrics.TotalPenaltyPoints += 10
		rep.LastSeen = now

	case EventTypeSecurity:
		rep.Metrics.SecurityEvents++
		rep.Metrics.ProtocolViolations++
		rep.Metrics.LastViolation = now
		rep.Metrics.ViolationStreak++
		rep.Metrics.TotalPenaltyPoints += 20 // Critical security event
		rep.LastSeen = now

	case EventTypeMisbehavior:
		rep.Metrics.ProtocolViolations++
		rep.Metrics.LastViolation = now
		rep.Metrics.ViolationStreak++
		rep.Metrics.TotalPenaltyPoints += 5
		rep.LastSeen = now

	case EventTypeLatencyMeasured:
		if event.Data.Latency > 0 {
			s.updateLatencyStats(rep, event.Data.Latency)
			rep.LastSeen = now
		}
	}

	// Recalculate score after event
	newScore := s.CalculateScore(rep)

	// Record score snapshot if significant change
	if len(rep.Metrics.RecentScores) == 0 || math.Abs(newScore-rep.Score) > 5.0 {
		snapshot := ScoreSnapshot{
			Timestamp: now,
			Score:     newScore,
			Reason:    event.EventType.String(),
		}
		rep.Metrics.RecentScores = append(rep.Metrics.RecentScores, snapshot)

		// Keep only last 100 snapshots
		if len(rep.Metrics.RecentScores) > 100 {
			rep.Metrics.RecentScores = rep.Metrics.RecentScores[len(rep.Metrics.RecentScores)-100:]
		}
	}

	rep.Score = newScore
	rep.TrustLevel = CalculateTrustLevel(newScore, rep.BanStatus.IsWhitelisted)
}

// updateLatencyStats updates latency statistics
func (s *Scorer) updateLatencyStats(rep *PeerReputation, latency time.Duration) {
	rep.Metrics.LatencyMeasurements++

	// Update average
	totalLatency := rep.Metrics.AvgResponseLatency * time.Duration(rep.Metrics.LatencyMeasurements-1)
	rep.Metrics.AvgResponseLatency = (totalLatency + latency) / time.Duration(rep.Metrics.LatencyMeasurements)

	// Update min/max
	if rep.Metrics.MinResponseLatency == 0 || latency < rep.Metrics.MinResponseLatency {
		rep.Metrics.MinResponseLatency = latency
	}
	if latency > rep.Metrics.MaxResponseLatency {
		rep.Metrics.MaxResponseLatency = latency
	}
}

// ShouldBan determines if a peer should be banned based on score and violations
func (s *Scorer) ShouldBan(rep *PeerReputation) (shouldBan bool, banType BanType, reason string) {
	// Check for severe violations - immediate permanent ban
	if rep.Metrics.DoubleSignAttempts > 0 {
		return true, BanTypePermanent, "double signing attempt detected"
	}

	if rep.Metrics.InvalidBlockProps >= 3 {
		return true, BanTypePermanent, "multiple invalid block proposals"
	}

	// Check for spam - temporary ban
	if rep.Metrics.SpamAttempts >= 5 {
		return true, BanTypeTemporary, "excessive spam attempts"
	}

	// Check for oversized messages - temporary ban for DoS attempts
	if rep.Metrics.OversizedMessages >= 3 {
		return true, BanTypeTemporary, "excessive oversized messages"
	}

	// Check score-based banning
	if rep.Score < 20.0 {
		// Multiple protocol violations with low score
		if rep.Metrics.ProtocolViolations >= 10 {
			return true, BanTypePermanent, "persistent protocol violations with low reputation"
		}
		// Low score from poor behavior
		return true, BanTypeTemporary, "reputation score below threshold"
	}

	// Check message validity
	if rep.Metrics.TotalMessages >= 100 && rep.Metrics.ValidMessageRatio < 0.50 {
		return true, BanTypeTemporary, "majority of messages invalid"
	}

	return false, BanTypeNone, ""
}

// GetBanDuration returns ban duration for temporary bans
func (s *Scorer) GetBanDuration(rep *PeerReputation) time.Duration {
	// Base duration: 1 hour
	baseDuration := 1 * time.Hour

	// Increase duration based on ban count (exponential backoff)
	multiplier := math.Pow(2, float64(rep.BanStatus.BanCount))
	duration := time.Duration(float64(baseDuration) * multiplier)

	// Cap at 7 days
	maxDuration := 7 * 24 * time.Hour
	if duration > maxDuration {
		duration = maxDuration
	}

	return duration
}
