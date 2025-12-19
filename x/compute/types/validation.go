package types

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const (
	// Maximum lengths for string fields
	MaxMonikerLength        = 128
	MaxEndpointLength       = 256
	MaxContainerImageLength = 512
	MaxCommandLength        = 1024
	MaxEnvVarKeyLength      = 128
	MaxEnvVarValueLength    = 1024
	MaxOutputURLLength      = 512
	MaxOutputHashLength     = 128

	// Maximum array sizes
	MaxEnvVarsCount     = 100
	MaxCommandArgsCount = 50

	// Container image validation
	MaxImageTagLength = 128
)

var (
	// Allowed registries for container images (whitelist)
	AllowedRegistries = []string{
		"docker.io",
		"gcr.io",
		"ghcr.io",
		"quay.io",
		"registry.hub.docker.com",
	}

	// Allowed URL schemes for output URLs
	AllowedURLSchemes = []string{
		"https",
		"ipfs",
		"ar", // Arweave
	}

	// Regular expressions for validation
	containerImageRegex = regexp.MustCompile(`^([a-z0-9]+([._-][a-z0-9]+)*(/[a-z0-9]+([._-][a-z0-9]+)*)*)(:([a-zA-Z0-9._-]+))?$`)
	envVarKeyRegex      = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	hashRegex           = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

// ValidateContainerImage validates and sanitizes a container image string
func ValidateContainerImage(image string) error {
	if image == "" {
		return fmt.Errorf("container image cannot be empty")
	}

	if len(image) > MaxContainerImageLength {
		return fmt.Errorf("container image exceeds maximum length of %d characters", MaxContainerImageLength)
	}

	// Remove any whitespace
	image = strings.TrimSpace(image)

	// Validate format
	if !containerImageRegex.MatchString(image) {
		return fmt.Errorf("invalid container image format: %s", image)
	}

	// Extract registry
	parts := strings.SplitN(image, "/", 2)
	var registry string
	if len(parts) == 2 && strings.Contains(parts[0], ".") {
		registry = parts[0]
	} else {
		registry = "docker.io" // Default registry
	}

	// Validate against whitelist
	registryAllowed := false
	for _, allowed := range AllowedRegistries {
		if registry == allowed || strings.HasSuffix(registry, "."+allowed) {
			registryAllowed = true
			break
		}
	}

	if !registryAllowed {
		return fmt.Errorf("registry %s is not in the allowed whitelist", registry)
	}

	// Check for suspicious patterns
	suspiciousPatterns := []string{
		"..", // Path traversal
		"//", // Double slashes
		"\\", // Backslashes
		" ",  // Spaces (should be trimmed)
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(image, pattern) {
			return fmt.Errorf("container image contains suspicious pattern: %s", pattern)
		}
	}

	return nil
}

// ValidateOutputURL validates and sanitizes an output URL
func ValidateOutputURL(outputURL string) error {
	if outputURL == "" {
		return fmt.Errorf("output URL cannot be empty")
	}

	if len(outputURL) > MaxOutputURLLength {
		return fmt.Errorf("output URL exceeds maximum length of %d characters", MaxOutputURLLength)
	}

	// Parse URL
	parsedURL, err := url.Parse(outputURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Validate scheme against whitelist
	schemeAllowed := false
	for _, allowed := range AllowedURLSchemes {
		if parsedURL.Scheme == allowed {
			schemeAllowed = true
			break
		}
	}

	if !schemeAllowed {
		return fmt.Errorf("URL scheme %s is not allowed (must be one of: %v)", parsedURL.Scheme, AllowedURLSchemes)
	}

	// Additional validation for HTTPS URLs
	if parsedURL.Scheme == "https" {
		// Ensure host is present
		if parsedURL.Host == "" {
			return fmt.Errorf("HTTPS URL must have a valid host")
		}

		// Block localhost and private IP ranges
		blockedHosts := []string{
			"localhost",
			"127.0.0.1",
			"0.0.0.0",
			"::1",
		}
		for _, blocked := range blockedHosts {
			if strings.Contains(parsedURL.Host, blocked) {
				return fmt.Errorf("URL host %s is blocked", parsedURL.Host)
			}
		}

		// Block private IP ranges (basic check)
		if strings.HasPrefix(parsedURL.Host, "10.") ||
			strings.HasPrefix(parsedURL.Host, "192.168.") ||
			strings.HasPrefix(parsedURL.Host, "172.16.") {
			return fmt.Errorf("private IP addresses are not allowed")
		}
	}

	// Check for suspicious patterns
	if strings.Contains(outputURL, "..") {
		return fmt.Errorf("URL contains path traversal pattern")
	}

	return nil
}

// ValidateOutputHash validates an output hash
func ValidateOutputHash(hash string) error {
	if hash == "" {
		return fmt.Errorf("output hash cannot be empty")
	}

	if len(hash) > MaxOutputHashLength {
		return fmt.Errorf("output hash exceeds maximum length of %d characters", MaxOutputHashLength)
	}

	// Validate hash format (expects SHA-256 hex string)
	if !hashRegex.MatchString(hash) {
		return fmt.Errorf("invalid hash format (expected 64 hex characters)")
	}

	return nil
}

// ValidateMoniker validates a provider moniker
func ValidateMoniker(moniker string) error {
	if moniker == "" {
		return fmt.Errorf("moniker cannot be empty")
	}

	if len(moniker) > MaxMonikerLength {
		return fmt.Errorf("moniker exceeds maximum length of %d characters", MaxMonikerLength)
	}

	// Check for control characters
	for _, r := range moniker {
		if r < 32 || r == 127 {
			return fmt.Errorf("moniker contains invalid control characters")
		}
	}

	return nil
}

// ValidateEndpoint validates a provider endpoint URL
func ValidateEndpoint(endpoint string) error {
	if endpoint == "" {
		return fmt.Errorf("endpoint cannot be empty")
	}

	if len(endpoint) > MaxEndpointLength {
		return fmt.Errorf("endpoint exceeds maximum length of %d characters", MaxEndpointLength)
	}

	// Parse as URL
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid endpoint URL: %w", err)
	}

	// Must use HTTPS
	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return fmt.Errorf("endpoint must use HTTP or HTTPS scheme")
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("endpoint must have a valid host")
	}

	return nil
}

// ValidateCommand validates command arguments
func ValidateCommand(command []string) error {
	if len(command) > MaxCommandArgsCount {
		return fmt.Errorf("command has too many arguments (max %d)", MaxCommandArgsCount)
	}

	totalLength := 0
	for _, arg := range command {
		totalLength += len(arg)

		// Check for null bytes
		if strings.Contains(arg, "\x00") {
			return fmt.Errorf("command argument contains null byte")
		}

		// Check for suspicious shell metacharacters
		suspiciousChars := []string{";", "|", "&", "`", "$", ">", "<"}
		for _, char := range suspiciousChars {
			if strings.Contains(arg, char) {
				return fmt.Errorf("command argument contains potentially dangerous character: %s", char)
			}
		}
	}

	if totalLength > MaxCommandLength {
		return fmt.Errorf("total command length exceeds maximum of %d characters", MaxCommandLength)
	}

	return nil
}

// ValidateEnvVars validates environment variables
func ValidateEnvVars(envVars map[string]string) error {
	if len(envVars) > MaxEnvVarsCount {
		return fmt.Errorf("too many environment variables (max %d)", MaxEnvVarsCount)
	}

	for key, value := range envVars {
		// Validate key length
		if len(key) > MaxEnvVarKeyLength {
			return fmt.Errorf("environment variable key %s exceeds maximum length of %d", key, MaxEnvVarKeyLength)
		}

		// Validate key format (uppercase letters, numbers, and underscores)
		if !envVarKeyRegex.MatchString(key) {
			return fmt.Errorf("invalid environment variable key: %s (must start with letter, contain only uppercase letters, numbers, and underscores)", key)
		}

		// Validate value length
		if len(value) > MaxEnvVarValueLength {
			return fmt.Errorf("environment variable value for %s exceeds maximum length of %d", key, MaxEnvVarValueLength)
		}

		// Check for null bytes in value
		if strings.Contains(value, "\x00") {
			return fmt.Errorf("environment variable value for %s contains null byte", key)
		}

		// Block certain sensitive environment variables
		blockedKeys := []string{
			"LD_PRELOAD",
			"LD_LIBRARY_PATH",
			"PATH", // Be careful with PATH modifications
		}
		for _, blocked := range blockedKeys {
			if key == blocked {
				return fmt.Errorf("environment variable %s is not allowed", key)
			}
		}
	}

	return nil
}

// SanitizeString removes control characters and trims whitespace
func SanitizeString(s string) string {
	// Remove control characters
	var sanitized strings.Builder
	for _, r := range s {
		if r >= 32 && r != 127 {
			sanitized.WriteRune(r)
		}
	}

	// Trim whitespace
	return strings.TrimSpace(sanitized.String())
}
