package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthManager manages API authentication
type AuthManager struct {
	apiKeys      map[string]*APIKey
	jwtSecret    []byte
	mu           sync.RWMutex
	cache        CacheInterface
}

// APIKey represents an API key
type APIKey struct {
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Tier        string    `json:"tier"` // "free", "pro", "enterprise"
	RateLimit   int       `json:"rate_limit"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Owner       string    `json:"owner"`
	Permissions []string  `json:"permissions"`
}

// JWTClaims represents JWT claims
type JWTClaims struct {
	jwt.RegisteredClaims
	APIKey      string   `json:"api_key"`
	Tier        string   `json:"tier"`
	Permissions []string `json:"permissions"`
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(jwtSecret string, cache CacheInterface) *AuthManager {
	secret := []byte(jwtSecret)
	if len(secret) == 0 {
		// Generate random secret if none provided
		secret = make([]byte, 32)
		rand.Read(secret)
	}

	am := &AuthManager{
		apiKeys:   make(map[string]*APIKey),
		jwtSecret: secret,
		cache:     cache,
	}

	// Load API keys from database or cache
	am.loadAPIKeys()

	return am
}

// loadAPIKeys loads API keys from storage
func (am *AuthManager) loadAPIKeys() {
	// In production, load from database
	// For now, create some sample keys
	am.mu.Lock()
	defer am.mu.Unlock()

	// Sample free tier key
	am.apiKeys["free_key_12345"] = &APIKey{
		Key:         "free_key_12345",
		Name:        "Free Tier Demo",
		Tier:        "free",
		RateLimit:   100, // 100 requests per minute
		Active:      true,
		CreatedAt:   time.Now(),
		Owner:       "demo@github.com",
		Permissions: []string{"read"},
	}

	// Sample pro tier key
	am.apiKeys["pro_key_67890"] = &APIKey{
		Key:         "pro_key_67890",
		Name:        "Pro Tier Demo",
		Tier:        "pro",
		RateLimit:   1000, // 1000 requests per minute
		Active:      true,
		CreatedAt:   time.Now(),
		Owner:       "pro@github.com",
		Permissions: []string{"read", "write"},
	}
}

// ValidateAPIKey validates an API key
func (am *AuthManager) ValidateAPIKey(key string) (*APIKey, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	apiKey, exists := am.apiKeys[key]
	if !exists {
		return nil, fmt.Errorf("invalid API key")
	}

	if !apiKey.Active {
		return nil, fmt.Errorf("API key is inactive")
	}

	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return nil, fmt.Errorf("API key has expired")
	}

	return apiKey, nil
}

// GenerateAPIKey generates a new API key
func (am *AuthManager) GenerateAPIKey(name, tier, owner string, permissions []string) (*APIKey, error) {
	// Generate random key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	key := hex.EncodeToString(keyBytes)

	// Determine rate limit based on tier
	rateLimit := 100 // free
	switch tier {
	case "pro":
		rateLimit = 1000
	case "enterprise":
		rateLimit = 10000
	}

	apiKey := &APIKey{
		Key:         key,
		Name:        name,
		Tier:        tier,
		RateLimit:   rateLimit,
		Active:      true,
		CreatedAt:   time.Now(),
		Owner:       owner,
		Permissions: permissions,
	}

	am.mu.Lock()
	am.apiKeys[key] = apiKey
	am.mu.Unlock()

	// In production, save to database

	return apiKey, nil
}

// RevokeAPIKey revokes an API key
func (am *AuthManager) RevokeAPIKey(key string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	apiKey, exists := am.apiKeys[key]
	if !exists {
		return fmt.Errorf("API key not found")
	}

	apiKey.Active = false

	// In production, update database

	return nil
}

// GenerateJWT generates a JWT token for an API key
func (am *AuthManager) GenerateJWT(apiKey *APIKey, duration time.Duration) (string, error) {
	claims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "paw-explorer",
		},
		APIKey:      apiKey.Key,
		Tier:        apiKey.Tier,
		Permissions: apiKey.Permissions,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(am.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateJWT validates a JWT token
func (am *AuthManager) ValidateJWT(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return am.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// AuthMiddleware creates an authentication middleware
func AuthMiddleware(authManager *AuthManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for API key in header
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			key, err := authManager.ValidateAPIKey(apiKey)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "invalid_api_key",
					"message": err.Error(),
				})
				c.Abort()
				return
			}

			// Store API key info in context
			c.Set("api_key", key)
			c.Set("tier", key.Tier)
			c.Set("permissions", key.Permissions)
			c.Next()
			return
		}

		// Check for JWT token in Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// Extract token from "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "invalid_auth_header",
					"message": "Authorization header must be in format: Bearer <token>",
				})
				c.Abort()
				return
			}

			claims, err := authManager.ValidateJWT(parts[1])
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "invalid_token",
					"message": err.Error(),
				})
				c.Abort()
				return
			}

			// Store claims in context
			c.Set("jwt_claims", claims)
			c.Set("tier", claims.Tier)
			c.Set("permissions", claims.Permissions)
			c.Next()
			return
		}

		// No authentication provided - allow with default limits
		c.Set("tier", "anonymous")
		c.Set("permissions", []string{"read"})
		c.Next()
	}
}

// RequirePermission creates a middleware that requires a specific permission
func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		permissions, exists := c.Get("permissions")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "permission_denied",
				"message": "No permissions found",
			})
			c.Abort()
			return
		}

		permList, ok := permissions.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "permission_denied",
				"message": "Invalid permissions format",
			})
			c.Abort()
			return
		}

		// Check if required permission is in the list
		hasPermission := false
		for _, p := range permList {
			if p == permission || p == "*" {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "permission_denied",
				"message": fmt.Sprintf("Permission '%s' is required", permission),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireTier creates a middleware that requires a minimum tier
func RequireTier(minTier string) gin.HandlerFunc {
	tierLevels := map[string]int{
		"anonymous":  0,
		"free":       1,
		"pro":        2,
		"enterprise": 3,
	}

	return func(c *gin.Context) {
		tierValue, exists := c.Get("tier")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "tier_required",
				"message": "No tier information found",
			})
			c.Abort()
			return
		}

		tier, ok := tierValue.(string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "tier_required",
				"message": "Invalid tier format",
			})
			c.Abort()
			return
		}

		currentLevel := tierLevels[tier]
		requiredLevel := tierLevels[minTier]

		if currentLevel < requiredLevel {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "tier_required",
				"message": fmt.Sprintf("Minimum tier '%s' is required", minTier),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// HashAPIKey creates a secure hash of an API key for storage
func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// GetAPIKeyInfo returns info about an API key (without sensitive data)
func (am *AuthManager) GetAPIKeyInfo(key string) map[string]interface{} {
	am.mu.RLock()
	defer am.mu.RUnlock()

	apiKey, exists := am.apiKeys[key]
	if !exists {
		return nil
	}

	return map[string]interface{}{
		"name":        apiKey.Name,
		"tier":        apiKey.Tier,
		"rate_limit":  apiKey.RateLimit,
		"created_at":  apiKey.CreatedAt,
		"permissions": apiKey.Permissions,
		"active":      apiKey.Active,
	}
}

// ListAPIKeys returns all API keys for an owner
func (am *AuthManager) ListAPIKeys(owner string) []*APIKey {
	am.mu.RLock()
	defer am.mu.RUnlock()

	keys := make([]*APIKey, 0)
	for _, apiKey := range am.apiKeys {
		if apiKey.Owner == owner {
			keys = append(keys, apiKey)
		}
	}

	return keys
}

// UpdateAPIKey updates an existing API key
func (am *AuthManager) UpdateAPIKey(key string, updates map[string]interface{}) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	apiKey, exists := am.apiKeys[key]
	if !exists {
		return fmt.Errorf("API key not found")
	}

	// Update fields
	if name, ok := updates["name"].(string); ok {
		apiKey.Name = name
	}
	if active, ok := updates["active"].(bool); ok {
		apiKey.Active = active
	}
	if permissions, ok := updates["permissions"].([]string); ok {
		apiKey.Permissions = permissions
	}

	// In production, update database

	return nil
}
