package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/control-center/admin-api/middleware"
	"github.com/paw-chain/paw/control-center/admin-api/types"
)

// MockAuditLogger implements the AuditLogger interface for testing
type MockAuditLogger struct {
	logs []AuditEntry
}

type AuditEntry struct {
	UserID    string
	Username  string
	Action    string
	Resource  string
	IPAddress string
	Details   map[string]interface{}
	Success   bool
	Error     error
}

func (m *MockAuditLogger) LogAction(userID, username, action, resource, ipAddress string, details map[string]interface{}, success bool, err error) {
	m.logs = append(m.logs, AuditEntry{
		UserID:    userID,
		Username:  username,
		Action:    action,
		Resource:  resource,
		IPAddress: ipAddress,
		Details:   details,
		Success:   success,
		Error:     err,
	})
}

func TestAuthService_Authenticate(t *testing.T) {
	auditLogger := &MockAuditLogger{}
	authService := middleware.NewAuthService("test-secret", 30*time.Minute, auditLogger)

	tests := []struct {
		name     string
		username string
		password string
		wantErr  bool
	}{
		{
			name:     "valid admin credentials",
			username: "admin",
			password: "admin123",
			wantErr:  false,
		},
		{
			name:     "valid operator credentials",
			username: "operator",
			password: "operator123",
			wantErr:  false,
		},
		{
			name:     "invalid username",
			username: "nonexistent",
			password: "password",
			wantErr:  true,
		},
		{
			name:     "invalid password",
			username: "admin",
			password: "wrongpassword",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := authService.Authenticate(tt.username, tt.password)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.username, user.Username)
			}
		})
	}
}

func TestAuthService_GenerateToken(t *testing.T) {
	auditLogger := &MockAuditLogger{}
	authService := middleware.NewAuthService("test-secret", 30*time.Minute, auditLogger)

	user := &types.User{
		ID:       "test-user-001",
		Username: "testuser",
		Role:     types.RoleAdmin,
		Active:   true,
	}

	token, err := authService.GenerateToken(user)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Validate the generated token
	claims, err := authService.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Username, claims.Username)
	assert.Equal(t, user.Role, claims.Role)
}

func TestAuthService_ValidateToken_Expired(t *testing.T) {
	auditLogger := &MockAuditLogger{}
	// Create service with very short token duration
	authService := middleware.NewAuthService("test-secret", 1*time.Millisecond, auditLogger)

	user := &types.User{
		ID:       "test-user-001",
		Username: "testuser",
		Role:     types.RoleAdmin,
		Active:   true,
	}

	token, err := authService.GenerateToken(user)
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(2 * time.Millisecond)

	// Token should be expired
	_, err = authService.ValidateToken(token)
	assert.Error(t, err)
}

func TestAuthMiddleware_Success(t *testing.T) {
	auditLogger := &MockAuditLogger{}
	authService := middleware.NewAuthService("test-secret", 30*time.Minute, auditLogger)

	user := &types.User{
		ID:       "test-user-001",
		Username: "testuser",
		Role:     types.RoleAdmin,
		Active:   true,
	}

	token, err := authService.GenerateToken(user)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(authService.AuthMiddleware())
	router.GET("/test", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
		})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), user.ID)
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	auditLogger := &MockAuditLogger{}
	authService := middleware.NewAuthService("test-secret", 30*time.Minute, auditLogger)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(authService.AuthMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Contains(t, resp.Body.String(), "unauthorized")
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	auditLogger := &MockAuditLogger{}
	authService := middleware.NewAuthService("test-secret", 30*time.Minute, auditLogger)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(authService.AuthMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Contains(t, resp.Body.String(), "unauthorized")
}

func TestRequireRole_Success(t *testing.T) {
	auditLogger := &MockAuditLogger{}
	authService := middleware.NewAuthService("test-secret", 30*time.Minute, auditLogger)

	user := &types.User{
		ID:       "admin-001",
		Username: "admin",
		Role:     types.RoleAdmin,
		Active:   true,
	}

	token, err := authService.GenerateToken(user)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(authService.AuthMiddleware())
	router.Use(authService.RequireRole(types.RoleOperator)) // Admin should pass Operator requirement
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestRequireRole_InsufficientPermissions(t *testing.T) {
	auditLogger := &MockAuditLogger{}
	authService := middleware.NewAuthService("test-secret", 30*time.Minute, auditLogger)

	user := &types.User{
		ID:       "readonly-001",
		Username: "readonly",
		Role:     types.RoleReadOnly,
		Active:   true,
	}

	token, err := authService.GenerateToken(user)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(authService.AuthMiddleware())
	router.Use(authService.RequireRole(types.RoleAdmin)) // ReadOnly should fail Admin requirement
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusForbidden, resp.Code)
	assert.Contains(t, resp.Body.String(), "forbidden")
}

func TestRequirePermission_Success(t *testing.T) {
	auditLogger := &MockAuditLogger{}
	authService := middleware.NewAuthService("test-secret", 30*time.Minute, auditLogger)

	user := &types.User{
		ID:       "admin-001",
		Username: "admin",
		Role:     types.RoleAdmin,
		Active:   true,
	}

	token, err := authService.GenerateToken(user)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(authService.AuthMiddleware())
	router.Use(authService.RequirePermission(types.PermissionUpdateParams)) // Admin has this permission
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("POST", "/test", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestRequirePermission_InsufficientPermissions(t *testing.T) {
	auditLogger := &MockAuditLogger{}
	authService := middleware.NewAuthService("test-secret", 30*time.Minute, auditLogger)

	user := &types.User{
		ID:       "readonly-001",
		Username: "readonly",
		Role:     types.RoleReadOnly,
		Active:   true,
	}

	token, err := authService.GenerateToken(user)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(authService.AuthMiddleware())
	router.Use(authService.RequirePermission(types.PermissionUpdateParams)) // ReadOnly doesn't have this
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("POST", "/test", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusForbidden, resp.Code)
	assert.Contains(t, resp.Body.String(), "forbidden")
}

func TestSessionManagement(t *testing.T) {
	auditLogger := &MockAuditLogger{}
	authService := middleware.NewAuthService("test-secret", 30*time.Minute, auditLogger)

	// Create a session
	session, err := authService.CreateSession("user-001", "127.0.0.1", types.RoleAdmin)
	require.NoError(t, err)
	assert.NotEmpty(t, session.SessionID)

	// Validate the session
	validatedSession, err := authService.ValidateSession(session.SessionID)
	require.NoError(t, err)
	assert.Equal(t, session.UserID, validatedSession.UserID)
	assert.Equal(t, session.Role, validatedSession.Role)

	// Try to validate an invalid session
	_, err = authService.ValidateSession("invalid-session-id")
	assert.Error(t, err)
}
