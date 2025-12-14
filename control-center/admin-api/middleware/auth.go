package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/paw-chain/paw/control-center/admin-api/types"
)

// AuthService handles authentication and authorization
type AuthService struct {
	jwtSecret     []byte
	tokenDuration time.Duration
	sessions      map[string]*types.Session
	users         map[string]*types.User
	mu            sync.RWMutex
	auditLogger   AuditLogger
}

// AuditLogger interface for logging audit events
type AuditLogger interface {
	LogAction(userID, username, action, resource, ipAddress string, details map[string]interface{}, success bool, err error)
}

// JWTClaims represents JWT token claims
type JWTClaims struct {
	jwt.RegisteredClaims
	UserID   string     `json:"user_id"`
	Username string     `json:"username"`
	Role     types.Role `json:"role"`
}

// NewAuthService creates a new authentication service
func NewAuthService(jwtSecret string, tokenDuration time.Duration, auditLogger AuditLogger) *AuthService {
	if jwtSecret == "" {
		// Generate random secret if not provided
		secret := make([]byte, 32)
		rand.Read(secret)
		jwtSecret = hex.EncodeToString(secret)
	}

	as := &AuthService{
		jwtSecret:     []byte(jwtSecret),
		tokenDuration: tokenDuration,
		sessions:      make(map[string]*types.Session),
		users:         make(map[string]*types.User),
		auditLogger:   auditLogger,
	}

	// Create default admin user (for testing - remove in production)
	as.createDefaultUser()

	return as
}

// createDefaultUser creates a default admin user
func (as *AuthService) createDefaultUser() {
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)

	as.users["admin"] = &types.User{
		ID:           "admin-001",
		Username:     "admin",
		Email:        "admin@paw-chain.com",
		Role:         types.RoleAdmin,
		Active:       true,
		CreatedAt:    time.Now(),
		RequiresMFA:  false,
		PasswordHash: string(passwordHash),
	}

	// Create operator user
	operatorHash, _ := bcrypt.GenerateFromPassword([]byte("operator123"), bcrypt.DefaultCost)
	as.users["operator"] = &types.User{
		ID:           "operator-001",
		Username:     "operator",
		Email:        "operator@paw-chain.com",
		Role:         types.RoleOperator,
		Active:       true,
		CreatedAt:    time.Now(),
		RequiresMFA:  false,
		PasswordHash: string(operatorHash),
	}

	// Create readonly user
	readonlyHash, _ := bcrypt.GenerateFromPassword([]byte("readonly123"), bcrypt.DefaultCost)
	as.users["readonly"] = &types.User{
		ID:           "readonly-001",
		Username:     "readonly",
		Email:        "readonly@paw-chain.com",
		Role:         types.RoleReadOnly,
		Active:       true,
		CreatedAt:    time.Now(),
		RequiresMFA:  false,
		PasswordHash: string(readonlyHash),
	}
}

// GenerateToken generates a JWT token for a user
func (as *AuthService) GenerateToken(user *types.User) (string, error) {
	claims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(as.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "paw-admin-api",
			Subject:   user.ID,
		},
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(as.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func (as *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return as.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// Authenticate authenticates a user with username and password
func (as *AuthService) Authenticate(username, password string) (*types.User, error) {
	as.mu.RLock()
	user, exists := as.users[username]
	as.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("invalid credentials")
	}

	if !user.Active {
		return nil, fmt.Errorf("account is inactive")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Update last login
	as.mu.Lock()
	user.LastLoginAt = time.Now()
	as.mu.Unlock()

	return user, nil
}

// CreateSession creates a new session for a user
func (as *AuthService) CreateSession(userID, ipAddress string, role types.Role) (*types.Session, error) {
	sessionID := generateSessionID()

	session := &types.Session{
		SessionID: sessionID,
		UserID:    userID,
		Role:      role,
		ExpiresAt: time.Now().Add(as.tokenDuration),
		IPAddress: ipAddress,
	}

	as.mu.Lock()
	as.sessions[sessionID] = session
	as.mu.Unlock()

	return session, nil
}

// ValidateSession validates a session ID
func (as *AuthService) ValidateSession(sessionID string) (*types.Session, error) {
	as.mu.RLock()
	session, exists := as.sessions[sessionID]
	as.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("invalid session")
	}

	if time.Now().After(session.ExpiresAt) {
		as.mu.Lock()
		delete(as.sessions, sessionID)
		as.mu.Unlock()
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

// GetUser retrieves a user by ID
func (as *AuthService) GetUser(userID string) (*types.User, error) {
	as.mu.RLock()
	defer as.mu.RUnlock()

	for _, user := range as.users {
		if user.ID == userID {
			return user, nil
		}
	}

	return nil, fmt.Errorf("user not found")
}

// AuthMiddleware creates a middleware that validates JWT tokens
func (as *AuthService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			as.auditLogger.LogAction("", "", "auth_failure", "api", c.ClientIP(), map[string]interface{}{
				"reason": "missing_auth_header",
			}, false, fmt.Errorf("missing authorization header"))

			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Missing authorization header",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			as.auditLogger.LogAction("", "", "auth_failure", "api", c.ClientIP(), map[string]interface{}{
				"reason": "invalid_auth_header_format",
			}, false, fmt.Errorf("invalid authorization header format"))

			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		claims, err := as.ValidateToken(parts[1])
		if err != nil {
			as.auditLogger.LogAction("", "", "auth_failure", "api", c.ClientIP(), map[string]interface{}{
				"reason": "invalid_token",
				"error":  err.Error(),
			}, false, err)

			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid token",
			})
			c.Abort()
			return
		}

		// Store claims in context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// RequireRole creates a middleware that requires a specific role
func (as *AuthService) RequireRole(requiredRole types.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "No role information found",
			})
			c.Abort()
			return
		}

		role, ok := roleValue.(types.Role)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "Invalid role format",
			})
			c.Abort()
			return
		}

		// Check if role meets requirement
		if !hasRequiredRole(role, requiredRole) {
			userID, _ := c.Get("user_id")
			username, _ := c.Get("username")

			as.auditLogger.LogAction(
				userID.(string),
				username.(string),
				"access_denied",
				c.Request.URL.Path,
				c.ClientIP(),
				map[string]interface{}{
					"required_role": requiredRole,
					"user_role":     role,
				},
				false,
				fmt.Errorf("insufficient permissions"),
			)

			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": fmt.Sprintf("Role '%s' is required", requiredRole),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermission creates a middleware that requires a specific permission
func (as *AuthService) RequirePermission(permission types.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "No role information found",
			})
			c.Abort()
			return
		}

		role, ok := roleValue.(types.Role)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "Invalid role format",
			})
			c.Abort()
			return
		}

		if !role.HasPermission(permission) {
			userID, _ := c.Get("user_id")
			username, _ := c.Get("username")

			as.auditLogger.LogAction(
				userID.(string),
				username.(string),
				"permission_denied",
				c.Request.URL.Path,
				c.ClientIP(),
				map[string]interface{}{
					"required_permission": permission,
					"user_role":           role,
				},
				false,
				fmt.Errorf("insufficient permissions"),
			)

			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": fmt.Sprintf("Permission '%s' is required", permission),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// hasRequiredRole checks if a user's role meets the required role
func hasRequiredRole(userRole, requiredRole types.Role) bool {
	// Role hierarchy: superuser > admin > operator > readonly
	roleLevel := map[types.Role]int{
		types.RoleReadOnly:  1,
		types.RoleOperator:  2,
		types.RoleAdmin:     3,
		types.RoleSuperUser: 4,
	}

	return roleLevel[userRole] >= roleLevel[requiredRole]
}

// generateSessionID generates a random session ID
func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
