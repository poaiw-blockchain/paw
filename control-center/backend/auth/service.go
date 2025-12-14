package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/paw-chain/paw/control-center/backend/audit"
	"golang.org/x/crypto/bcrypt"
)

// Role represents user roles in the system
type Role string

const (
	RoleViewer     Role = "Viewer"
	RoleOperator   Role = "Operator"
	RoleAdmin      Role = "Admin"
	RoleSuperAdmin Role = "SuperAdmin"
)

// User represents a system user
type User struct {
	ID           uint      `json:"id"`
	Email        string    `json:"email"`
	Password     string    `json:"-"` // Never expose in JSON
	Role         Role      `json:"role"`
	TwoFactorKey string    `json:"-"` // 2FA secret key
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	LastLoginAt  *time.Time `json:"last_login_at"`
}

// Claims represents JWT token claims
type Claims struct {
	Email       string   `json:"email"`
	Role        Role     `json:"role"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// Service provides authentication and authorization services
type Service struct {
	jwtSecret       []byte
	tokenExpiration time.Duration
	auditService    *audit.Service
	users           map[string]*User // In-memory user store (replace with DB in production)
	sessions        map[string]*Session // Active sessions
}

// Session represents an active user session
type Session struct {
	Token     string
	User      *User
	ExpiresAt time.Time
	IPAddress string
}

// NewService creates a new authentication service
func NewService(jwtSecret string, tokenExpiration time.Duration, auditService *audit.Service) *Service {
	svc := &Service{
		jwtSecret:       []byte(jwtSecret),
		tokenExpiration: tokenExpiration,
		auditService:    auditService,
		users:           make(map[string]*User),
		sessions:        make(map[string]*Session),
	}

	// Initialize with default admin user
	svc.initializeDefaultUsers()

	return svc
}

// initializeDefaultUsers creates default system users
func (s *Service) initializeDefaultUsers() {
	// Default SuperAdmin (password: admin123 - CHANGE IN PRODUCTION)
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	s.users["admin@paw.network"] = &User{
		ID:        1,
		Email:     "admin@paw.network",
		Password:  string(hashedPassword),
		Role:      RoleSuperAdmin,
		Enabled:   true,
		CreatedAt: time.Now(),
	}

	// Default Admin
	hashedPassword, _ = bcrypt.GenerateFromPassword([]byte("operator123"), bcrypt.DefaultCost)
	s.users["operator@paw.network"] = &User{
		ID:        2,
		Email:     "operator@paw.network",
		Password:  string(hashedPassword),
		Role:      RoleAdmin,
		Enabled:   true,
		CreatedAt: time.Now(),
	}
}

// Login handles user login
func (s *Service) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Find user
	user, exists := s.users[req.Email]
	if !exists || !user.Enabled {
		s.auditService.Log(audit.Entry{
			User:      req.Email,
			Action:    "login_failed",
			Result:    "user_not_found",
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
		})
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		s.auditService.Log(audit.Entry{
			User:      req.Email,
			Action:    "login_failed",
			Result:    "invalid_password",
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
		})
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate JWT token
	token, err := s.generateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Update last login
	now := time.Now()
	user.LastLoginAt = &now

	// Store session
	sessionID := generateSessionID()
	s.sessions[sessionID] = &Session{
		Token:     token,
		User:      user,
		ExpiresAt: time.Now().Add(s.tokenExpiration),
		IPAddress: c.ClientIP(),
	}

	// Log successful login
	s.auditService.Log(audit.Entry{
		User:      req.Email,
		Role:      string(user.Role),
		Action:    "login_success",
		Result:    "authenticated",
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		SessionID: sessionID,
	})

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"expires_at": time.Now().Add(s.tokenExpiration).Unix(),
		"user": gin.H{
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

// RefreshToken refreshes an expired token
func (s *Service) RefreshToken(c *gin.Context) {
	// Extract token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization header"})
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// Parse token (without validation to allow expired tokens)
	token, _ := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})

	if claims, ok := token.Claims.(*Claims); ok {
		user, exists := s.users[claims.Email]
		if !exists || !user.Enabled {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			return
		}

		// Generate new token
		newToken, err := s.generateToken(user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token":      newToken,
			"expires_at": time.Now().Add(s.tokenExpiration).Unix(),
		})
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
	}
}

// AuthMiddleware validates JWT tokens
func (s *Service) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization header"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return s.jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(*Claims); ok {
			// Store user info in context
			c.Set("user_email", claims.Email)
			c.Set("user_role", claims.Role)
			c.Set("user_permissions", claims.Permissions)
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
		}
	}
}

// RoleMiddleware checks if user has required role
func (s *Service) RoleMiddleware(requiredRole Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "Role not found in context"})
			c.Abort()
			return
		}

		role := userRole.(Role)

		// Role hierarchy: SuperAdmin > Admin > Operator > Viewer
		if !s.hasPermission(role, requiredRole) {
			c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Requires %s role or higher", requiredRole)})
			c.Abort()
			return
		}

		c.Next()
	}
}

// TwoFactorMiddleware validates 2FA code for critical operations
func (s *Service) TwoFactorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get 2FA code from header
		twoFactorCode := c.GetHeader("X-2FA-Code")
		if twoFactorCode == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "2FA code required for this operation"})
			c.Abort()
			return
		}

		// Validate 2FA code
		userEmail, _ := c.Get("user_email")
		user := s.users[userEmail.(string)]

		if !s.validate2FA(user, twoFactorCode) {
			s.auditService.Log(audit.Entry{
				User:      userEmail.(string),
				Action:    "2fa_failed",
				Result:    "invalid_code",
				IPAddress: c.ClientIP(),
			})
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid 2FA code"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// WSAuthMiddleware validates authentication for WebSocket connections
func (s *Service) WSAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from query parameter (WebSocket can't send custom headers easily)
		tokenString := c.Query("token")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
			c.Abort()
			return
		}

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return s.jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(*Claims); ok {
			c.Set("user_email", claims.Email)
			c.Set("user_role", claims.Role)
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
		}
	}
}

// generateToken creates a new JWT token
func (s *Service) generateToken(user *User) (string, error) {
	permissions := s.getPermissions(user.Role)

	claims := Claims{
		Email:       user.Email,
		Role:        user.Role,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "paw-control-center",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// getPermissions returns permissions for a role
func (s *Service) getPermissions(role Role) []string {
	switch role {
	case RoleSuperAdmin:
		return []string{
			"read:all", "write:all", "admin:all",
			"control:circuit-breaker", "control:emergency",
			"manage:users",
		}
	case RoleAdmin:
		return []string{
			"read:all", "write:params", "write:alerts",
			"control:circuit-breaker",
		}
	case RoleOperator:
		return []string{
			"read:all", "write:tests",
		}
	case RoleViewer:
		return []string{
			"read:all",
		}
	default:
		return []string{}
	}
}

// hasPermission checks if a role has required permission
func (s *Service) hasPermission(userRole, requiredRole Role) bool {
	roleLevel := map[Role]int{
		RoleViewer:     1,
		RoleOperator:   2,
		RoleAdmin:      3,
		RoleSuperAdmin: 4,
	}

	return roleLevel[userRole] >= roleLevel[requiredRole]
}

// validate2FA validates a 2FA code
func (s *Service) validate2FA(user *User, code string) bool {
	// Simplified 2FA validation
	// In production, use TOTP (Time-based One-Time Password) library
	if user.TwoFactorKey == "" {
		return true // 2FA not enabled
	}

	// For demo purposes, accept "123456" as valid code
	return code == "123456"
}

// generateSessionID generates a random session ID
func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// CreateUser creates a new user (SuperAdmin only)
func (s *Service) CreateUser(email string, password string, role Role) error {
	if _, exists := s.users[email]; exists {
		return errors.New("user already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	s.users[email] = &User{
		ID:        uint(len(s.users) + 1),
		Email:     email,
		Password:  string(hashedPassword),
		Role:      role,
		Enabled:   true,
		CreatedAt: time.Now(),
	}

	return nil
}

// GetUser retrieves a user by email
func (s *Service) GetUser(email string) (*User, error) {
	user, exists := s.users[email]
	if !exists {
		return nil, errors.New("user not found")
	}
	return user, nil
}

// ListUsers returns all users
func (s *Service) ListUsers() []*User {
	users := make([]*User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	return users
}

// UpdateUser updates a user's role or status
func (s *Service) UpdateUser(email string, role *Role, enabled *bool) error {
	user, exists := s.users[email]
	if !exists {
		return errors.New("user not found")
	}

	if role != nil {
		user.Role = *role
	}

	if enabled != nil {
		user.Enabled = *enabled
	}

	return nil
}

// DeleteUser deletes a user
func (s *Service) DeleteUser(email string) error {
	if _, exists := s.users[email]; !exists {
		return errors.New("user not found")
	}

	delete(s.users, email)
	return nil
}
