package keeper

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/paw-chain/paw/x/compute/types"
)

// RateLimiter implements a token bucket rate limiter for gRPC queries
type RateLimiter struct {
	buckets         map[string]*tokenBucket
	mu              sync.RWMutex
	rate            int // tokens per second
	burst           int // max bucket size
	cleanupInterval time.Duration
}

// tokenBucket represents a token bucket for rate limiting
type tokenBucket struct {
	tokens    float64
	lastCheck time.Time
	mu        sync.Mutex
}

// NewRateLimiter creates a new rate limiter
// rate: number of requests allowed per second
// burst: maximum burst size
func NewRateLimiter(rate, burst int) *RateLimiter {
	rl := &RateLimiter{
		buckets:         make(map[string]*tokenBucket),
		rate:            rate,
		burst:           burst,
		cleanupInterval: 5 * time.Minute,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if a request from the given client should be allowed
func (rl *RateLimiter) Allow(clientID string) bool {
	rl.mu.Lock()
	bucket, exists := rl.buckets[clientID]
	if !exists {
		bucket = &tokenBucket{
			tokens:    float64(rl.burst),
			lastCheck: time.Now(),
		}
		rl.buckets[clientID] = bucket
	}
	rl.mu.Unlock()

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(bucket.lastCheck).Seconds()

	// Add tokens based on elapsed time
	bucket.tokens += elapsed * float64(rl.rate)
	if bucket.tokens > float64(rl.burst) {
		bucket.tokens = float64(rl.burst)
	}

	bucket.lastCheck = now

	// Check if we have at least one token
	if bucket.tokens >= 1.0 {
		bucket.tokens -= 1.0
		return true
	}

	return false
}

// cleanup periodically removes old buckets
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for clientID, bucket := range rl.buckets {
			bucket.mu.Lock()
			if now.Sub(bucket.lastCheck) > rl.cleanupInterval {
				delete(rl.buckets, clientID)
			}
			bucket.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// getClientID extracts a client identifier from the context
// Priority: metadata > peer IP
func getClientID(ctx context.Context) string {
	// Try to get from metadata (e.g., API key, user ID)
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if clientIDs := md.Get("x-client-id"); len(clientIDs) > 0 {
			return clientIDs[0]
		}
		if apiKeys := md.Get("x-api-key"); len(apiKeys) > 0 {
			return apiKeys[0]
		}
	}

	// Fall back to peer IP address
	if p, ok := peer.FromContext(ctx); ok {
		return p.Addr.String()
	}

	return "unknown"
}

// RateLimitedQueryServer wraps a query server with rate limiting
type RateLimitedQueryServer struct {
	types.QueryServer
	limiter *RateLimiter
}

// NewRateLimitedQueryServer creates a new rate-limited query server
func NewRateLimitedQueryServer(qs types.QueryServer, limiter *RateLimiter) *RateLimitedQueryServer {
	return &RateLimitedQueryServer{
		QueryServer: qs,
		limiter:     limiter,
	}
}

// checkRateLimit checks rate limit and returns error if exceeded
func (rlqs *RateLimitedQueryServer) checkRateLimit(ctx context.Context) error {
	clientID := getClientID(ctx)
	if !rlqs.limiter.Allow(clientID) {
		return status.Errorf(
			codes.ResourceExhausted,
			"query rate limit exceeded",
		)
	}
	return nil
}

// Params wraps the Params query with rate limiting
func (rlqs *RateLimitedQueryServer) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if err := rlqs.checkRateLimit(ctx); err != nil {
		return nil, fmt.Errorf("Params: rate limit: %w", err)
	}
	return rlqs.QueryServer.Params(ctx, req)
}

// Provider wraps the Provider query with rate limiting
func (rlqs *RateLimitedQueryServer) Provider(ctx context.Context, req *types.QueryProviderRequest) (*types.QueryProviderResponse, error) {
	if err := rlqs.checkRateLimit(ctx); err != nil {
		return nil, fmt.Errorf("Provider: rate limit: %w", err)
	}
	return rlqs.QueryServer.Provider(ctx, req)
}

// Providers wraps the Providers query with rate limiting
func (rlqs *RateLimitedQueryServer) Providers(ctx context.Context, req *types.QueryProvidersRequest) (*types.QueryProvidersResponse, error) {
	if err := rlqs.checkRateLimit(ctx); err != nil {
		return nil, fmt.Errorf("Providers: rate limit: %w", err)
	}
	return rlqs.QueryServer.Providers(ctx, req)
}

// ActiveProviders wraps the ActiveProviders query with rate limiting
func (rlqs *RateLimitedQueryServer) ActiveProviders(ctx context.Context, req *types.QueryActiveProvidersRequest) (*types.QueryActiveProvidersResponse, error) {
	if err := rlqs.checkRateLimit(ctx); err != nil {
		return nil, fmt.Errorf("ActiveProviders: rate limit: %w", err)
	}
	return rlqs.QueryServer.ActiveProviders(ctx, req)
}

// Request wraps the Request query with rate limiting
func (rlqs *RateLimitedQueryServer) Request(ctx context.Context, req *types.QueryRequestRequest) (*types.QueryRequestResponse, error) {
	if err := rlqs.checkRateLimit(ctx); err != nil {
		return nil, fmt.Errorf("Request: rate limit: %w", err)
	}
	return rlqs.QueryServer.Request(ctx, req)
}

// Requests wraps the Requests query with rate limiting
func (rlqs *RateLimitedQueryServer) Requests(ctx context.Context, req *types.QueryRequestsRequest) (*types.QueryRequestsResponse, error) {
	if err := rlqs.checkRateLimit(ctx); err != nil {
		return nil, fmt.Errorf("Requests: rate limit: %w", err)
	}
	return rlqs.QueryServer.Requests(ctx, req)
}

// RequestsByRequester wraps the RequestsByRequester query with rate limiting
func (rlqs *RateLimitedQueryServer) RequestsByRequester(ctx context.Context, req *types.QueryRequestsByRequesterRequest) (*types.QueryRequestsByRequesterResponse, error) {
	if err := rlqs.checkRateLimit(ctx); err != nil {
		return nil, fmt.Errorf("RequestsByRequester: rate limit: %w", err)
	}
	return rlqs.QueryServer.RequestsByRequester(ctx, req)
}

// RequestsByProvider wraps the RequestsByProvider query with rate limiting
func (rlqs *RateLimitedQueryServer) RequestsByProvider(ctx context.Context, req *types.QueryRequestsByProviderRequest) (*types.QueryRequestsByProviderResponse, error) {
	if err := rlqs.checkRateLimit(ctx); err != nil {
		return nil, fmt.Errorf("RequestsByProvider: rate limit: %w", err)
	}
	return rlqs.QueryServer.RequestsByProvider(ctx, req)
}

// RequestsByStatus wraps the RequestsByStatus query with rate limiting
func (rlqs *RateLimitedQueryServer) RequestsByStatus(ctx context.Context, req *types.QueryRequestsByStatusRequest) (*types.QueryRequestsByStatusResponse, error) {
	if err := rlqs.checkRateLimit(ctx); err != nil {
		return nil, fmt.Errorf("RequestsByStatus: rate limit: %w", err)
	}
	return rlqs.QueryServer.RequestsByStatus(ctx, req)
}

// Result wraps the Result query with rate limiting
func (rlqs *RateLimitedQueryServer) Result(ctx context.Context, req *types.QueryResultRequest) (*types.QueryResultResponse, error) {
	if err := rlqs.checkRateLimit(ctx); err != nil {
		return nil, fmt.Errorf("Result: rate limit: %w", err)
	}
	return rlqs.QueryServer.Result(ctx, req)
}

// EstimateCost wraps the EstimateCost query with rate limiting
func (rlqs *RateLimitedQueryServer) EstimateCost(ctx context.Context, req *types.QueryEstimateCostRequest) (*types.QueryEstimateCostResponse, error) {
	if err := rlqs.checkRateLimit(ctx); err != nil {
		return nil, fmt.Errorf("EstimateCost: rate limit: %w", err)
	}
	return rlqs.QueryServer.EstimateCost(ctx, req)
}
