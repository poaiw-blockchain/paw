package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/paw-chain/paw/control-center/admin-api/types"
)

// Client represents an Admin API client
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
	username   string
	password   string
}

// Config holds client configuration
type Config struct {
	BaseURL  string
	Username string
	Password string
	Timeout  time.Duration
}

// NewClient creates a new Admin API client
func NewClient(config *Config) (*Client, error) {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	client := &Client{
		baseURL:  config.BaseURL,
		username: config.Username,
		password: config.Password,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}

	// Authenticate on creation
	if err := client.authenticate(); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	return client, nil
}

// authenticate performs initial authentication
func (c *Client) authenticate() error {
	reqBody := map[string]string{
		"username": c.username,
		"password": c.password,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/api/v1/auth/login", c.baseURL),
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed: %s", string(body))
	}

	var authResp struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return err
	}

	c.token = authResp.Token
	return nil
}

// doRequest performs an authenticated HTTP request
func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", c.baseURL, path), reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// If unauthorized, try to re-authenticate and retry once
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		if err := c.authenticate(); err != nil {
			return nil, err
		}

		// Retry with new token
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
		return c.httpClient.Do(req)
	}

	return resp, nil
}

// GetModuleParams retrieves parameters for a specific module
func (c *Client) GetModuleParams(module string) (map[string]interface{}, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/v1/admin/params/%s", module), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get params: %s", string(body))
	}

	var result struct {
		Module string                 `json:"module"`
		Params map[string]interface{} `json:"params"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Params, nil
}

// UpdateModuleParams updates parameters for a specific module
func (c *Client) UpdateModuleParams(module string, params map[string]interface{}, reason string) error {
	reqBody := types.ParamUpdate{
		Module: module,
		Params: params,
		Reason: reason,
	}

	resp, err := c.doRequest("POST", fmt.Sprintf("/api/v1/admin/params/%s", module), reqBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update params: %s", string(body))
	}

	return nil
}

// ResetModuleParams resets module parameters to defaults
func (c *Client) ResetModuleParams(module string, reason string) error {
	reqBody := map[string]string{
		"reason": reason,
	}

	resp, err := c.doRequest("POST", fmt.Sprintf("/api/v1/admin/params/%s/reset", module), reqBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to reset params: %s", string(body))
	}

	return nil
}

// GetParamHistory retrieves parameter change history
func (c *Client) GetParamHistory(module string, limit, offset int) ([]*types.ParamHistoryEntry, error) {
	path := fmt.Sprintf("/api/v1/admin/params/history?module=%s&limit=%d&offset=%d", module, limit, offset)
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get history: %s", string(body))
	}

	var result struct {
		History []*types.ParamHistoryEntry `json:"history"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.History, nil
}

// PauseModule pauses a specific module via circuit breaker
func (c *Client) PauseModule(module string, reason string, autoResume bool) error {
	reqBody := map[string]interface{}{
		"reason":      reason,
		"auto_resume": autoResume,
	}

	resp, err := c.doRequest("POST", fmt.Sprintf("/api/v1/admin/circuit-breaker/%s/pause", module), reqBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to pause module: %s", string(body))
	}

	return nil
}

// ResumeModule resumes a paused module
func (c *Client) ResumeModule(module string, reason string) error {
	reqBody := map[string]string{
		"reason": reason,
	}

	resp, err := c.doRequest("POST", fmt.Sprintf("/api/v1/admin/circuit-breaker/%s/resume", module), reqBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to resume module: %s", string(body))
	}

	return nil
}

// GetCircuitBreakerStatus retrieves circuit breaker status
func (c *Client) GetCircuitBreakerStatus(module string) (*types.CircuitBreakerStatus, error) {
	path := "/api/v1/admin/circuit-breaker/status"
	if module != "" {
		path += "?module=" + module
	}

	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get status: %s", string(body))
	}

	if module != "" {
		var result struct {
			Status *types.CircuitBreakerStatus `json:"status"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}
		return result.Status, nil
	}

	var result struct {
		Statuses []*types.CircuitBreakerStatus `json:"statuses"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Statuses) > 0 {
		return result.Statuses[0], nil
	}

	return nil, fmt.Errorf("no status found")
}

// EmergencyPauseDEX performs emergency pause of DEX module
func (c *Client) EmergencyPauseDEX(reason string, mfaCode string) error {
	return c.emergencyPause("pause-dex", reason, mfaCode)
}

// EmergencyPauseOracle performs emergency pause of Oracle module
func (c *Client) EmergencyPauseOracle(reason string, mfaCode string) error {
	return c.emergencyPause("pause-oracle", reason, mfaCode)
}

// EmergencyPauseCompute performs emergency pause of Compute module
func (c *Client) EmergencyPauseCompute(reason string, mfaCode string) error {
	return c.emergencyPause("pause-compute", reason, mfaCode)
}

// emergencyPause is a helper for emergency pause operations
func (c *Client) emergencyPause(action string, reason string, mfaCode string) error {
	reqBody := map[string]string{
		"reason":   reason,
		"mfa_code": mfaCode,
	}

	resp, err := c.doRequest("POST", fmt.Sprintf("/api/v1/admin/emergency/%s", action), reqBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("emergency operation failed: %s", string(body))
	}

	return nil
}

// EmergencyResumeAll resumes all paused modules
func (c *Client) EmergencyResumeAll(reason string, mfaCode string) error {
	reqBody := map[string]string{
		"reason":   reason,
		"mfa_code": mfaCode,
	}

	resp, err := c.doRequest("POST", "/api/v1/admin/emergency/resume-all", reqBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("emergency resume failed: %s", string(body))
	}

	return nil
}

// ScheduleUpgrade schedules a network upgrade
func (c *Client) ScheduleUpgrade(name string, height int64, info string) error {
	reqBody := types.UpgradeSchedule{
		Name:   name,
		Height: height,
		Info:   info,
	}

	resp, err := c.doRequest("POST", "/api/v1/admin/upgrade/schedule", reqBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to schedule upgrade: %s", string(body))
	}

	return nil
}

// CancelUpgrade cancels a scheduled upgrade
func (c *Client) CancelUpgrade(name string, reason string) error {
	reqBody := map[string]string{
		"name":   name,
		"reason": reason,
	}

	resp, err := c.doRequest("POST", "/api/v1/admin/upgrade/cancel", reqBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to cancel upgrade: %s", string(body))
	}

	return nil
}

// GetUpgradeStatus retrieves upgrade status
func (c *Client) GetUpgradeStatus(name string) (*types.UpgradeSchedule, error) {
	path := "/api/v1/admin/upgrade/status"
	if name != "" {
		path += "?name=" + name
	}

	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get upgrade status: %s", string(body))
	}

	if name != "" {
		var result struct {
			Schedule *types.UpgradeSchedule `json:"schedule"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}
		return result.Schedule, nil
	}

	var result struct {
		Schedules []*types.UpgradeSchedule `json:"schedules"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Schedules) > 0 {
		return result.Schedules[0], nil
	}

	return nil, fmt.Errorf("no upgrade found")
}

// RefreshToken refreshes the authentication token
func (c *Client) RefreshToken() error {
	resp, err := c.doRequest("POST", "/api/v1/auth/refresh", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to refresh token")
	}

	var result struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	c.token = result.Token
	return nil
}

// Logout logs out the current session
func (c *Client) Logout() error {
	resp, err := c.doRequest("POST", "/api/v1/auth/logout", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	c.token = ""
	return nil
}
