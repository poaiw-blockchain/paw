package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/go-bip39"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication logic
type AuthService struct {
	jwtSecret     []byte
	users         map[string]*User         // In-memory user store (use database in production)
	refreshTokens map[string]*RefreshToken // In-memory refresh token store
	revokedTokens map[string]time.Time     // Token revocation list (JTI -> expiration time)
	mu            sync.RWMutex
}

// RefreshToken represents a refresh token
type RefreshToken struct {
	Token     string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// NewAuthService creates a new authentication service
func NewAuthService(jwtSecret []byte) *AuthService {
	as := &AuthService{
		jwtSecret:     jwtSecret,
		users:         make(map[string]*User),
		refreshTokens: make(map[string]*RefreshToken),
		revokedTokens: make(map[string]time.Time),
	}

	// Start background goroutine to clean up expired revoked tokens
	go as.cleanupRevokedTokens()

	return as
}

// Claims represents JWT claims
type Claims struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Address   string `json:"address"`
	TokenType string `json:"token_type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// handleRegister handles user registration with BIP39 wallet creation
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

	// Generate user ID
	userID := generateUserID()

	var mnemonic string
	var address string

	// Check if user wants to recover from mnemonic or create new wallet
	if req.Recover && req.Mnemonic != "" {
		// Recover wallet from existing mnemonic
		// Clean up mnemonic
		mnemonic = strings.TrimSpace(req.Mnemonic)
		words := strings.Fields(mnemonic)
		mnemonic = strings.Join(words, " ")

		// Validate mnemonic
		if !bip39.IsMnemonicValid(mnemonic) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid mnemonic",
				Details: "The provided mnemonic phrase is invalid or has incorrect checksum",
			})
			return
		}

		// Verify word count
		wordCount := len(words)
		if wordCount != 12 && wordCount != 24 {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid mnemonic length",
				Details: fmt.Sprintf("Expected 12 or 24 words, got %d", wordCount),
			})
			return
		}

		// Create key from mnemonic
		hdPath := hd.CreateHDPath(sdk.GetConfig().GetCoinType(), 0, 0)
		keyInfo, err := s.clientCtx.Keyring.NewAccount(
			req.Username,
			mnemonic,
			keyring.DefaultBIP39Passphrase,
			hdPath.String(),
			hd.Secp256k1,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Failed to recover wallet from mnemonic",
				Details: err.Error(),
			})
			return
		}

		// Get address
		addr, err := keyInfo.GetAddress()
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Failed to get address from key",
				Details: err.Error(),
			})
			return
		}
		address = addr.String()

		// Don't return the mnemonic for recovery (user already has it)
		mnemonic = ""

	} else {
		// Create new wallet with BIP39 mnemonic
		// Generate entropy (256 bits for 24-word mnemonic)
		entropy := make([]byte, 32)
		if _, err := rand.Read(entropy); err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Failed to generate secure entropy",
				Details: err.Error(),
			})
			return
		}

		// Generate mnemonic from entropy
		mnemonic, err = bip39.NewMnemonic(entropy)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Failed to generate mnemonic",
				Details: err.Error(),
			})
			return
		}

		// Validate the generated mnemonic
		if !bip39.IsMnemonicValid(mnemonic) {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Generated mnemonic failed validation",
				Details: "Internal error during mnemonic generation",
			})
			return
		}

		// Create key from mnemonic
		hdPath := hd.CreateHDPath(sdk.GetConfig().GetCoinType(), 0, 0)
		keyInfo, err := s.clientCtx.Keyring.NewAccount(
			req.Username,
			mnemonic,
			keyring.DefaultBIP39Passphrase,
			hdPath.String(),
			hd.Secp256k1,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Failed to create wallet",
				Details: err.Error(),
			})
			return
		}

		// Get address
		addr, err := keyInfo.GetAddress()
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Failed to get address from key",
				Details: err.Error(),
			})
			return
		}
		address = addr.String()
	}

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

	// Prepare response
	response := gin.H{
		"user_id":  userID,
		"username": req.Username,
		"address":  address,
		"success":  true,
		"message":  "Registration successful",
	}

	// Include mnemonic only for new wallet creation (not recovery)
	if !req.Recover && mnemonic != "" {
		response["mnemonic"] = mnemonic
		response["warning"] = "IMPORTANT: Save your mnemonic phrase securely. It's the only way to recover your wallet."
	}

	c.JSON(http.StatusCreated, response)
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

	// Generate access token (15 minutes) and refresh token (7 days)
	accessToken, refreshToken, expiresIn, err := s.authService.GenerateTokenPair(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to generate tokens",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		Username:     user.Username,
		UserID:       user.ID,
		Address:      user.Address,
	})
}

// GenerateTokenPair generates both access and refresh tokens
func (as *AuthService) GenerateTokenPair(user *User) (accessToken string, refreshToken string, expiresIn int64, err error) {
	// Generate access token (15 minutes)
	accessToken, err = as.generateToken(user, "access", 15*time.Minute)
	if err != nil {
		return "", "", 0, err
	}

	// Generate refresh token (7 days)
	refreshToken, err = as.generateToken(user, "refresh", 7*24*time.Hour)
	if err != nil {
		return "", "", 0, err
	}

	// Store refresh token
	as.mu.Lock()
	as.refreshTokens[refreshToken] = &RefreshToken{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}
	as.mu.Unlock()

	return accessToken, refreshToken, 900, nil // 900 seconds = 15 minutes
}

// generateToken generates a JWT token with specified type and duration
func (as *AuthService) generateToken(user *User, tokenType string, duration time.Duration) (string, error) {
	now := time.Now()
	expirationTime := now.Add(duration)

	// Generate unique JTI for token revocation
	jti := generateJTI()

	claims := &Claims{
		UserID:    user.ID,
		Username:  user.Username,
		Address:   user.Address,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "paw-api",
			Subject:   user.ID,
			ID:        jti, // Unique token ID for revocation
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(as.jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// generateJTI generates a unique token ID
func generateJTI() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ValidateToken validates a JWT token and returns the claims
func (as *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method to prevent algorithm substitution attacks
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

	// Check if token has been revoked
	as.mu.RLock()
	_, revoked := as.revokedTokens[claims.ID]
	as.mu.RUnlock()

	if revoked {
		return nil, fmt.Errorf("token has been revoked")
	}

	// Validate token type
	if claims.TokenType != "access" && claims.TokenType != "refresh" {
		return nil, fmt.Errorf("invalid token type")
	}

	return claims, nil
}

// RefreshAccessToken generates a new access token using a refresh token
func (as *AuthService) RefreshAccessToken(refreshTokenString string) (string, int64, error) {
	// Validate refresh token
	claims, err := as.ValidateToken(refreshTokenString)
	if err != nil {
		return "", 0, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Verify it's a refresh token
	if claims.TokenType != "refresh" {
		return "", 0, fmt.Errorf("not a refresh token")
	}

	// Check if refresh token exists and is not expired
	as.mu.RLock()
	storedToken, exists := as.refreshTokens[refreshTokenString]
	as.mu.RUnlock()

	if !exists {
		return "", 0, fmt.Errorf("refresh token not found")
	}

	if time.Now().After(storedToken.ExpiresAt) {
		// Clean up expired token
		as.mu.Lock()
		delete(as.refreshTokens, refreshTokenString)
		as.mu.Unlock()
		return "", 0, fmt.Errorf("refresh token expired")
	}

	// Get user
	as.mu.RLock()
	var user *User
	for _, u := range as.users {
		if u.ID == claims.UserID {
			user = u
			break
		}
	}
	as.mu.RUnlock()

	if user == nil {
		return "", 0, fmt.Errorf("user not found")
	}

	// Generate new access token
	accessToken, err := as.generateToken(user, "access", 15*time.Minute)
	if err != nil {
		return "", 0, err
	}

	return accessToken, 900, nil // 900 seconds = 15 minutes
}

// RevokeToken revokes a token by adding its JTI to the revocation list
func (as *AuthService) RevokeToken(tokenString string) error {
	claims, err := as.ValidateToken(tokenString)
	if err != nil {
		// If token is invalid, consider it already revoked
		return nil
	}

	// Add to revocation list with expiration time
	as.mu.Lock()
	as.revokedTokens[claims.ID] = claims.ExpiresAt.Time
	as.mu.Unlock()

	// If it's a refresh token, remove it from refresh tokens map
	if claims.TokenType == "refresh" {
		as.mu.Lock()
		delete(as.refreshTokens, tokenString)
		as.mu.Unlock()
	}

	return nil
}

// cleanupRevokedTokens periodically removes expired tokens from revocation list
func (as *AuthService) cleanupRevokedTokens() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		as.mu.Lock()
		for jti, expiresAt := range as.revokedTokens {
			if now.After(expiresAt) {
				delete(as.revokedTokens, jti)
			}
		}

		// Also clean up expired refresh tokens
		for token, rt := range as.refreshTokens {
			if now.After(rt.ExpiresAt) {
				delete(as.refreshTokens, token)
			}
		}
		as.mu.Unlock()
	}
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

// handleRefreshToken handles token refresh requests
func (s *Server) handleRefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Details: err.Error(),
		})
		return
	}

	// Generate new access token using refresh token
	accessToken, expiresIn, err := s.authService.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Failed to refresh token",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, RefreshTokenResponse{
		Token:     accessToken,
		ExpiresIn: expiresIn,
	})
}

// handleLogout handles user logout (token revocation)
func (s *Server) handleLogout(c *gin.Context) {
	// Get token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Authorization header required",
		})
		return
	}

	// Extract token (remove "Bearer " prefix)
	token := ""
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		token = authHeader[7:]
	} else {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid authorization header format",
		})
		return
	}

	// Revoke the token
	if err := s.authService.RevokeToken(token); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to logout",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Logged out successfully",
	})
}
