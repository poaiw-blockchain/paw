package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestServer creates a test server instance
func setupTestServer(t *testing.T) *Server {
	// Create codec
	interfaceRegistry := types.NewInterfaceRegistry()
	marshaler := codec.NewProtoCodec(interfaceRegistry)

	// Create client context
	clientCtx := client.Context{}.
		WithCodec(marshaler).
		WithInterfaceRegistry(interfaceRegistry)

	// Create test config
	config := &Config{
		Host:         "localhost",
		Port:         "5000",
		ChainID:      "paw-test",
		NodeURI:      "tcp://localhost:26657",
		JWTSecret:    []byte("test-secret"),
		CORSOrigins:  []string{"*"},
		RateLimitRPS: 1000,
	}

	// Create server
	server, err := NewServer(clientCtx, config)
	require.NoError(t, err)

	return server
}

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	server := setupTestServer(t)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
	assert.NotNil(t, response["timestamp"])
}

// TestUserRegistration tests user registration
func TestUserRegistration(t *testing.T) {
	server := setupTestServer(t)

	tests := []struct {
		name           string
		payload        RegisterRequest
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "successful registration",
			payload: RegisterRequest{
				Username: "newuser",
				Password: "password123",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response SuccessResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.True(t, response.Success)
			},
		},
		{
			name: "username too short",
			payload: RegisterRequest{
				Username: "ab",
				Password: "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "password too short",
			payload: RegisterRequest{
				Username: "validuser",
				Password: "123",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

// TestUserLogin tests user login
func TestUserLogin(t *testing.T) {
	server := setupTestServer(t)

	// First register a user
	regPayload := RegisterRequest{
		Username: "logintest",
		Password: "password123",
	}
	body, _ := json.Marshal(regPayload)
	req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Test login
	loginPayload := LoginRequest{
		Username: "logintest",
		Password: "password123",
	}
	body, _ = json.Marshal(loginPayload)
	req, _ = http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response AuthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response.Token)
	assert.Equal(t, "logintest", response.Username)
	assert.NotEmpty(t, response.UserID)
}

// TestOrderCreation tests order creation
func TestOrderCreation(t *testing.T) {
	server := setupTestServer(t)

	// Register and login
	token := registerAndLogin(t, server, "trader", "password123")

	// Test order creation
	orderPayload := CreateOrderRequest{
		OrderType: "buy",
		Price:     10.50,
		Amount:    100,
	}
	body, _ := json.Marshal(orderPayload)
	req, _ := http.NewRequest("POST", "/api/orders/create", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response CreateOrderResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response.OrderID)
	assert.Equal(t, "buy", response.OrderType)
	assert.Equal(t, 10.50, response.Price)
	assert.Equal(t, 100.0, response.Amount)
}

// TestGetOrderBook tests order book retrieval
func TestGetOrderBook(t *testing.T) {
	server := setupTestServer(t)

	req, _ := http.NewRequest("GET", "/api/orders/book", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response OrderBook
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response.Bids)
	assert.NotNil(t, response.Asks)
}

// TestWalletBalance tests wallet balance retrieval
func TestWalletBalance(t *testing.T) {
	server := setupTestServer(t)

	// Register and login
	token := registerAndLogin(t, server, "walletuser", "password123")

	req, _ := http.NewRequest("GET", "/api/wallet/balance", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response BalanceResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response.Address)
	assert.GreaterOrEqual(t, response.AIXNBalance, 0.0)
}

// TestAtomicSwapPrepare tests atomic swap preparation
func TestAtomicSwapPrepare(t *testing.T) {
	server := setupTestServer(t)

	// Register and login
	token := registerAndLogin(t, server, "swapper", "password123")

	swapPayload := PrepareSwapRequest{
		CounterpartyAddress: "paw1counterparty123",
		SendAmount:          "1000000",
		SendDenom:           "paw",
		ReceiveAmount:       "500000",
		ReceiveDenom:        "usdc",
		TimeLockDuration:    3600,
	}
	body, _ := json.Marshal(swapPayload)
	req, _ := http.NewRequest("POST", "/api/atomic-swap/prepare", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response PrepareSwapResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response.SwapID)
	assert.NotEmpty(t, response.HashLock)
	assert.NotEmpty(t, response.Secret)
	assert.Equal(t, "pending", response.Status)
}

// TestGetPools tests pool listing
func TestGetPools(t *testing.T) {
	server := setupTestServer(t)

	req, _ := http.NewRequest("GET", "/api/pools", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	pools, ok := response["pools"].([]interface{})
	assert.True(t, ok)
	assert.Greater(t, len(pools), 0)
}

// TestAuthMiddleware tests authentication middleware
func TestAuthMiddleware(t *testing.T) {
	server := setupTestServer(t)

	// Test without token
	req, _ := http.NewRequest("GET", "/api/wallet/balance", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Test with invalid token
	req, _ = http.NewRequest("GET", "/api/wallet/balance", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w = httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Test with valid token
	token := registerAndLogin(t, server, "authtest", "password123")
	req, _ = http.NewRequest("GET", "/api/wallet/balance", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// Helper function to register and login a user
func registerAndLogin(t *testing.T, server *Server, username, password string) string {
	// Register
	regPayload := RegisterRequest{
		Username: username,
		Password: password,
	}
	body, _ := json.Marshal(regPayload)
	req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Login
	loginPayload := LoginRequest{
		Username: username,
		Password: password,
	}
	body, _ = json.Marshal(loginPayload)
	req, _ = http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var response AuthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	return response.Token
}
