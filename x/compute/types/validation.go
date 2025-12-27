package types

import (
	"fmt"
	"net"
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

		// SEC-3 FIX: Block ALL private IP ranges and IPv6 private addresses
		// This prevents SSRF attacks where providers could access internal services
		if isPrivateOrReservedIP(parsedURL.Host) {
			return fmt.Errorf("private or reserved IP addresses are not allowed")
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

	// Must use HTTPS in production; allow HTTP only for localhost/127.0.0.1 (development)
	if parsedURL.Scheme == "http" {
		host := parsedURL.Hostname()
		if host != "localhost" && host != "127.0.0.1" {
			return fmt.Errorf("endpoint must use HTTPS (HTTP only allowed for localhost/127.0.0.1)")
		}
	} else if parsedURL.Scheme != "https" {
		return fmt.Errorf("endpoint must use HTTPS scheme")
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

// isPrivateOrReservedIP checks if a host is a private or reserved IP address.
// SEC-3 FIX: This function provides comprehensive blocking of all private IP ranges
// to prevent SSRF (Server-Side Request Forgery) attacks.
//
// Blocked ranges:
// - IPv4: 10.0.0.0/8, 172.16.0.0/12 (172.16-31.x.x), 192.168.0.0/16, 169.254.0.0/16 (link-local)
// - IPv4: 127.0.0.0/8 (loopback), 0.0.0.0/8 (current network)
// - IPv6: ::1 (loopback), fe80::/10 (link-local), fc00::/7 (unique local)
// - IPv6: ::ffff:0:0/96 (IPv4-mapped)
func isPrivateOrReservedIP(host string) bool {
	// Strip port if present
	hostOnly := host
	if colonIdx := strings.LastIndex(host, ":"); colonIdx != -1 {
		// Check if this is an IPv6 address with brackets
		if strings.HasPrefix(host, "[") {
			// IPv6 with port: [::1]:8080
			if bracketIdx := strings.Index(host, "]"); bracketIdx != -1 {
				hostOnly = host[1:bracketIdx]
			}
		} else if strings.Count(host, ":") == 1 {
			// IPv4 with port: 192.168.1.1:8080
			hostOnly = host[:colonIdx]
		}
		// Otherwise it's an IPv6 without brackets, keep as-is
	}

	// Parse the IP address
	ip := net.ParseIP(hostOnly)
	if ip == nil {
		// Not an IP address (could be a hostname), check for string patterns
		// that might resolve to private addresses
		return false
	}

	// Check for loopback
	if ip.IsLoopback() {
		return true
	}

	// Check for private addresses
	if ip.IsPrivate() {
		return true
	}

	// Check for link-local addresses (169.254.x.x for IPv4, fe80::/10 for IPv6)
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// Check for unspecified address (0.0.0.0 or ::)
	if ip.IsUnspecified() {
		return true
	}

	// Additional IPv4 private range checks for net.IP limitations
	// 172.16.0.0/12 range: 172.16.0.0 - 172.31.255.255
	if ip4 := ip.To4(); ip4 != nil {
		// 172.16.0.0 - 172.31.255.255
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return true
		}
		// 100.64.0.0/10 (Carrier-Grade NAT)
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
		// 192.0.0.0/24 (IANA special purpose)
		if ip4[0] == 192 && ip4[1] == 0 && ip4[2] == 0 {
			return true
		}
		// 192.0.2.0/24 (TEST-NET-1)
		if ip4[0] == 192 && ip4[1] == 0 && ip4[2] == 2 {
			return true
		}
		// 198.51.100.0/24 (TEST-NET-2)
		if ip4[0] == 198 && ip4[1] == 51 && ip4[2] == 100 {
			return true
		}
		// 203.0.113.0/24 (TEST-NET-3)
		if ip4[0] == 203 && ip4[1] == 0 && ip4[2] == 113 {
			return true
		}
		// 224.0.0.0/4 (Multicast)
		if ip4[0] >= 224 && ip4[0] <= 239 {
			return true
		}
		// 240.0.0.0/4 (Reserved for future use)
		if ip4[0] >= 240 {
			return true
		}
	}

	// IPv6 unique local addresses (fc00::/7)
	if ip6 := ip.To16(); ip6 != nil && ip.To4() == nil {
		// fc00::/7 covers fc00:: through fdff::
		if ip6[0] == 0xfc || ip6[0] == 0xfd {
			return true
		}
		// IPv4-mapped IPv6 addresses (::ffff:0:0/96)
		// These could be used to bypass IPv4 checks
		if ip6[0] == 0 && ip6[1] == 0 && ip6[2] == 0 && ip6[3] == 0 &&
			ip6[4] == 0 && ip6[5] == 0 && ip6[6] == 0 && ip6[7] == 0 &&
			ip6[8] == 0 && ip6[9] == 0 && ip6[10] == 0xff && ip6[11] == 0xff {
			// This is an IPv4-mapped address, check the embedded IPv4
			embeddedIP := net.IPv4(ip6[12], ip6[13], ip6[14], ip6[15])
			return isPrivateOrReservedIP(embeddedIP.String())
		}
	}

	return false
}
