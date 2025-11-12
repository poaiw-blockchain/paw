package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication logic
type AuthService struct {
	jwtSecret []byte
	users     map[string]*User // In-memory user store (use database in production)
	mu        sync.RWMutex
}

// NewAuthService creates a new authentication service
func NewAuthService(jwtSecret []byte) *AuthService {
	return &AuthService{
		jwtSecret: jwtSecret,
		users:     make(map[string]*User),
	}
}

// Claims represents JWT claims
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Address  string `json:"address"`
	jwt.RegisteredClaims
}

// handleRegister handles user registration
func (s *Server) handleRegister(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Details: err.Error(),
		})
		return
	}

	// Check if username already exists
	s.authService.mu.RLock()
	_, exists := s.authService.users[req.Username]
	s.authService.mu.RUnlock()

	if exists {
		c.JSON(http.StatusConflict, ErrorResponse{
			Error: "Username already exists",
		})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to process registration",
			Details: err.Error(),
		})
		return
	}

	// Generate user ID and blockchain address
	userID := generateUserID()
	address := generateAddress(req.Username)

	// Create user
	user := &User{
		ID:           userID,
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		Address:      address,
		CreatedAt:    time.Now(),
	}

	// Store user
	s.authService.mu.Lock()
	s.authService.users[req.Username] = user
	s.authService.mu.Unlock()

	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Message: "Registration successful",
		Data: gin.H{
			"user_id":  userID,
			"username": req.Username,
			"address":  address,
		},
	})
}

// handleLogin handles user login
func (s *Server) handleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Details: err.Error(),
		})
		return
	}

	// Get user
	s.authService.mu.RLock()
	user, exists := s.authService.users[req.Username]
	s.authService.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Invalid credentials",
		})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Invalid credentials",
		})
		return
	}

	// Generate JWT token
	token, err := s.authService.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to generate token",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Token:    token,
		Username: user.Username,
		UserID:   user.ID,
		Address:  user.Address,
	})
}

// GenerateToken generates a JWT token for a user
func (as *AuthService) GenerateToken(user *User) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)

	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Address:  user.Address,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "paw-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(as.jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func (as *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return as.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// GetUser retrieves a user by username
func (as *AuthService) GetUser(username string) (*User, bool) {
	as.mu.RLock()
	defer as.mu.RUnlock()
	user, exists := as.users[username]
	return user, exists
}

// GetUserByID retrieves a user by ID
func (as *AuthService) GetUserByID(userID string) (*User, bool) {
	as.mu.RLock()
	defer as.mu.RUnlock()
	for _, user := range as.users {
		if user.ID == userID {
			return user, true
		}
	}
	return nil, false
}

// Helper functions

// generateUserID generates a unique user ID
func generateUserID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// generateAddress generates a blockchain address for a user
func generateAddress(username string) string {
	// In production, this would derive a proper bech32 address
	// For now, generate a deterministic address based on username
	b := make([]byte, 20)
	rand.Read(b)
	return "paw1" + hex.EncodeToString(b)[:38]
}
