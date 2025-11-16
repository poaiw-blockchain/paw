package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Secure version of authentication handlers with full validation

// handleRegister handles user registration with validation
func (s *Server) handleRegisterSecure(c *gin.Context) {
	var req RegisterRequest

	// Validate and bind JSON with size limit
	if err := ValidateAndBindJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Details: err.Error(),
			Code:    "INVALID_REQUEST",
		})
		return
	}

	// Validate request fields
	if err := ValidateRegisterRequest(&req); err != nil {
		if s.auditLogger != nil {
			s.auditLogger.LogAuthentication(c, req.Username, "", "failure", "validation failed: "+err.Error())
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Details: err.Error(),
			Code:    "VALIDATION_ERROR",
		})
		return
	}

	// Log registration attempt
	if s.auditLogger != nil {
		s.auditLogger.LogAuthentication(c, req.Username, "", "attempt", "registration attempt")
	}

	// Check if username already exists
	s.authService.mu.RLock()
	_, exists := s.authService.users[req.Username]
	s.authService.mu.RUnlock()

	if exists {
		if s.auditLogger != nil {
			s.auditLogger.LogAuthentication(c, req.Username, "", "failure", "username already exists")
		}
		c.JSON(http.StatusConflict, ErrorResponse{
			Error: "Username already exists",
			Code:  "USERNAME_EXISTS",
		})
		return
	}

	// Hash password with strong cost
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		if s.auditLogger != nil {
			s.auditLogger.LogAuthentication(c, req.Username, "", "failure", "password hashing failed")
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to process registration",
			Details: "Internal error",
			Code:    "INTERNAL_ERROR",
		})
		return
	}

	// Generate user ID and blockchain address
	userID := generateUserID()
	address := generateAddressSecure(req.Username)

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

	// Log successful registration
	if s.auditLogger != nil {
		s.auditLogger.LogAuthentication(c, req.Username, userID, "success", "registration successful")
	}

	// Sanitize output
	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Message: "Registration successful",
		Data: gin.H{
			"user_id":  SanitizeString(userID),
			"username": SanitizeString(req.Username),
			"address":  SanitizeString(address),
		},
	})
}

// handleLogin handles user login with validation
func (s *Server) handleLoginSecure(c *gin.Context) {
	var req LoginRequest

	// Validate and bind JSON with size limit
	if err := ValidateAndBindJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Details: err.Error(),
			Code:    "INVALID_REQUEST",
		})
		return
	}

	// Validate request fields
	if err := ValidateLoginRequest(&req); err != nil {
		if s.auditLogger != nil {
			s.auditLogger.LogAuthentication(c, req.Username, "", "failure", "validation failed: "+err.Error())
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Details: err.Error(),
			Code:    "VALIDATION_ERROR",
		})
		return
	}

	// Log login attempt
	if s.auditLogger != nil {
		s.auditLogger.LogAuthentication(c, req.Username, "", "attempt", "login attempt")
	}

	// Get user
	s.authService.mu.RLock()
	user, exists := s.authService.users[req.Username]
	s.authService.mu.RUnlock()

	if !exists {
		if s.auditLogger != nil {
			s.auditLogger.LogAuthentication(c, req.Username, "", "failure", "user not found")
		}
		// Use generic error message to prevent username enumeration
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Invalid credentials",
			Code:  "INVALID_CREDENTIALS",
		})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		if s.auditLogger != nil {
			s.auditLogger.LogAuthentication(c, req.Username, user.ID, "failure", "invalid password")
		}
		// Use generic error message
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Invalid credentials",
			Code:  "INVALID_CREDENTIALS",
		})
		return
	}

	// Generate access token (15 minutes) and refresh token (7 days)
	accessToken, refreshToken, expiresIn, err := s.authService.GenerateTokenPair(user)
	if err != nil {
		if s.auditLogger != nil {
			s.auditLogger.LogAuthentication(c, req.Username, user.ID, "failure", "token generation failed")
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to generate tokens",
			Details: "Internal error",
			Code:    "INTERNAL_ERROR",
		})
		return
	}

	// Log successful login
	if s.auditLogger != nil {
		s.auditLogger.LogAuthentication(c, req.Username, user.ID, "success", "login successful")
	}

	// Sanitize output
	c.JSON(http.StatusOK, AuthResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		Username:     SanitizeString(user.Username),
		UserID:       SanitizeString(user.ID),
		Address:      SanitizeString(user.Address),
	})
}

// handleRefreshToken handles token refresh with validation
func (s *Server) handleRefreshTokenSecure(c *gin.Context) {
	var req RefreshTokenRequest

	// Validate and bind JSON
	if err := ValidateAndBindJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Details: err.Error(),
			Code:    "INVALID_REQUEST",
		})
		return
	}

	// Validate refresh token format
	if req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Refresh token is required",
			Code:  "TOKEN_REQUIRED",
		})
		return
	}

	// Log refresh attempt
	if s.auditLogger != nil {
		s.auditLogger.LogSecurityEvent(
			"token_refresh",
			"info",
			"refresh_attempt",
			"attempt",
			fmt.Sprintf("Token refresh from IP: %s", c.ClientIP()),
			map[string]interface{}{
				"ip": c.ClientIP(),
			},
		)
	}

	// Generate new access token using refresh token
	accessToken, expiresIn, err := s.authService.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		if s.auditLogger != nil {
			s.auditLogger.LogSecurityEvent(
				"token_refresh",
				"warning",
				"refresh_failed",
				"failure",
				fmt.Sprintf("Token refresh failed from IP: %s", c.ClientIP()),
				map[string]interface{}{
					"ip":    c.ClientIP(),
					"error": err.Error(),
				},
			)
		}
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Failed to refresh token",
			Details: "Invalid or expired refresh token",
			Code:    "INVALID_REFRESH_TOKEN",
		})
		return
	}

	c.JSON(http.StatusOK, RefreshTokenResponse{
		Token:     accessToken,
		ExpiresIn: expiresIn,
	})
}

// handleLogout handles logout with validation
func (s *Server) handleLogoutSecure(c *gin.Context) {
	// Get user info from context (set by auth middleware)
	userID, username, _, err := GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	// Get token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Authorization header required",
			Code:  "AUTH_HEADER_REQUIRED",
		})
		return
	}

	// Extract token
	token := ""
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		token = authHeader[7:]
	} else {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid authorization header format",
			Code:  "INVALID_AUTH_HEADER",
		})
		return
	}

	// Revoke the token
	if err := s.authService.RevokeToken(token); err != nil {
		if s.auditLogger != nil {
			s.auditLogger.LogAuthentication(c, username, userID, "failure", "logout failed: "+err.Error())
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to logout",
			Details: "Internal error",
			Code:    "INTERNAL_ERROR",
		})
		return
	}

	// Log successful logout
	if s.auditLogger != nil {
		s.auditLogger.LogAuthentication(c, username, userID, "success", "logout successful")
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Logged out successfully",
	})
}

// generateAddressSecure generates a blockchain address securely
func generateAddressSecure(username string) string {
	// In production, derive proper bech32 address from HD wallet
	// For now, generate cryptographically secure random address
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		// Fallback to less secure method if crypto/rand fails
		for i := range b {
			b[i] = byte(time.Now().UnixNano() % 256)
		}
	}
	return "paw1" + hex.EncodeToString(b)[:38]
}

// Replace existing handlers - call these from routes
// These functions can be assigned to replace the existing handlers
// in the server initialization or route registration
