package fraud

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// Detector detects fraudulent faucet requests
type Detector struct {
	// IP tracking
	ipRequests map[string][]time.Time
	ipMu       sync.RWMutex

	// Address tracking
	addressRequests map[string][]time.Time
	addressMu       sync.RWMutex

	// Blacklist
	blacklistedIPs       map[string]time.Time
	blacklistedAddresses map[string]time.Time
	blacklistMu          sync.RWMutex

	// Fingerprint tracking (browser fingerprints, etc)
	fingerprints map[string][]time.Time
	fingerprintMu sync.RWMutex

	// Configuration
	maxRequestsPerIP      int
	maxRequestsPerAddress int
	timeWindow            time.Duration
	blacklistDuration     time.Duration
}

// FraudCheckResult represents the result of a fraud check
type FraudCheckResult struct {
	Allowed        bool     `json:"allowed"`
	Reason         string   `json:"reason,omitempty"`
	Score          float64  `json:"score"` // 0-1, higher is more suspicious
	Flags          []string `json:"flags,omitempty"`
	RemainingQuota int      `json:"remaining_quota,omitempty"`
}

// NewDetector creates a new fraud detector
func NewDetector() *Detector {
	d := &Detector{
		ipRequests:            make(map[string][]time.Time),
		addressRequests:       make(map[string][]time.Time),
		blacklistedIPs:        make(map[string]time.Time),
		blacklistedAddresses:  make(map[string]time.Time),
		fingerprints:          make(map[string][]time.Time),
		maxRequestsPerIP:      5,  // 5 requests
		maxRequestsPerAddress: 3,  // 3 requests per address
		timeWindow:            24 * time.Hour,
		blacklistDuration:     7 * 24 * time.Hour,
	}

	// Start cleanup goroutine
	go d.cleanup()

	return d
}

// CheckRequest checks if a request should be allowed
func (d *Detector) CheckRequest(ipAddress, recipientAddress, fingerprint string) *FraudCheckResult {
	result := &FraudCheckResult{
		Allowed: true,
		Score:   0.0,
		Flags:   make([]string, 0),
	}

	// Check blacklists
	if d.isIPBlacklisted(ipAddress) {
		result.Allowed = false
		result.Reason = "IP address is blacklisted"
		result.Score = 1.0
		result.Flags = append(result.Flags, "blacklisted_ip")
		return result
	}

	if d.isAddressBlacklisted(recipientAddress) {
		result.Allowed = false
		result.Reason = "Recipient address is blacklisted"
		result.Score = 1.0
		result.Flags = append(result.Flags, "blacklisted_address")
		return result
	}

	// Check IP rate limit
	ipCount := d.getIPRequestCount(ipAddress)
	if ipCount >= d.maxRequestsPerIP {
		result.Allowed = false
		result.Reason = fmt.Sprintf("Too many requests from this IP (%d/%d)", ipCount, d.maxRequestsPerIP)
		result.Score = 0.9
		result.Flags = append(result.Flags, "ip_rate_limit")
		return result
	}

	// Check address rate limit
	addressCount := d.getAddressRequestCount(recipientAddress)
	if addressCount >= d.maxRequestsPerAddress {
		result.Allowed = false
		result.Reason = fmt.Sprintf("Too many requests for this address (%d/%d)", addressCount, d.maxRequestsPerAddress)
		result.Score = 0.9
		result.Flags = append(result.Flags, "address_rate_limit")
		return result
	}

	// Check fingerprint (detect multiple addresses from same device)
	if fingerprint != "" {
		fingerprintCount := d.getFingerprintCount(fingerprint)
		if fingerprintCount >= 3 {
			result.Score += 0.5
			result.Flags = append(result.Flags, "suspicious_fingerprint")
		}
	}

	// Check for suspicious patterns
	if d.isSuspiciousIP(ipAddress) {
		result.Score += 0.3
		result.Flags = append(result.Flags, "suspicious_ip")
	}

	// Check for bot-like behavior
	if d.isBotLikeBehavior(ipAddress) {
		result.Score += 0.4
		result.Flags = append(result.Flags, "bot_like_behavior")
	}

	// If score is too high, block the request
	if result.Score >= 0.8 {
		result.Allowed = false
		result.Reason = "Suspicious activity detected"
	}

	result.RemainingQuota = d.maxRequestsPerIP - ipCount

	return result
}

// RecordRequest records a successful request
func (d *Detector) RecordRequest(ipAddress, recipientAddress, fingerprint string) {
	now := time.Now()

	// Record IP request
	d.ipMu.Lock()
	if _, exists := d.ipRequests[ipAddress]; !exists {
		d.ipRequests[ipAddress] = make([]time.Time, 0)
	}
	d.ipRequests[ipAddress] = append(d.ipRequests[ipAddress], now)
	d.ipMu.Unlock()

	// Record address request
	d.addressMu.Lock()
	if _, exists := d.addressRequests[recipientAddress]; !exists {
		d.addressRequests[recipientAddress] = make([]time.Time, 0)
	}
	d.addressRequests[recipientAddress] = append(d.addressRequests[recipientAddress], now)
	d.addressMu.Unlock()

	// Record fingerprint
	if fingerprint != "" {
		d.fingerprintMu.Lock()
		if _, exists := d.fingerprints[fingerprint]; !exists {
			d.fingerprints[fingerprint] = make([]time.Time, 0)
		}
		d.fingerprints[fingerprint] = append(d.fingerprints[fingerprint], now)
		d.fingerprintMu.Unlock()
	}

	log.Printf("Recorded request: IP=%s, Address=%s", ipAddress, recipientAddress)
}

// BlacklistIP adds an IP to the blacklist
func (d *Detector) BlacklistIP(ipAddress string, duration time.Duration) {
	d.blacklistMu.Lock()
	defer d.blacklistMu.Unlock()

	expiryTime := time.Now().Add(duration)
	d.blacklistedIPs[ipAddress] = expiryTime
	log.Printf("Blacklisted IP: %s until %s", ipAddress, expiryTime)
}

// BlacklistAddress adds an address to the blacklist
func (d *Detector) BlacklistAddress(address string, duration time.Duration) {
	d.blacklistMu.Lock()
	defer d.blacklistMu.Unlock()

	expiryTime := time.Now().Add(duration)
	d.blacklistedAddresses[address] = expiryTime
	log.Printf("Blacklisted address: %s until %s", address, expiryTime)
}

// isIPBlacklisted checks if an IP is blacklisted
func (d *Detector) isIPBlacklisted(ipAddress string) bool {
	d.blacklistMu.RLock()
	defer d.blacklistMu.RUnlock()

	expiryTime, exists := d.blacklistedIPs[ipAddress]
	if !exists {
		return false
	}

	// Check if blacklist has expired
	if time.Now().After(expiryTime) {
		// Cleanup will remove it later
		return false
	}

	return true
}

// isAddressBlacklisted checks if an address is blacklisted
func (d *Detector) isAddressBlacklisted(address string) bool {
	d.blacklistMu.RLock()
	defer d.blacklistMu.RUnlock()

	expiryTime, exists := d.blacklistedAddresses[address]
	if !exists {
		return false
	}

	if time.Now().After(expiryTime) {
		return false
	}

	return true
}

// getIPRequestCount gets the number of requests from an IP in the time window
func (d *Detector) getIPRequestCount(ipAddress string) int {
	d.ipMu.RLock()
	defer d.ipMu.RUnlock()

	requests, exists := d.ipRequests[ipAddress]
	if !exists {
		return 0
	}

	cutoff := time.Now().Add(-d.timeWindow)
	count := 0

	for _, t := range requests {
		if t.After(cutoff) {
			count++
		}
	}

	return count
}

// getAddressRequestCount gets the number of requests for an address in the time window
func (d *Detector) getAddressRequestCount(address string) int {
	d.addressMu.RLock()
	defer d.addressMu.RUnlock()

	requests, exists := d.addressRequests[address]
	if !exists {
		return 0
	}

	cutoff := time.Now().Add(-d.timeWindow)
	count := 0

	for _, t := range requests {
		if t.After(cutoff) {
			count++
		}
	}

	return count
}

// getFingerprintCount gets the number of requests with a fingerprint
func (d *Detector) getFingerprintCount(fingerprint string) int {
	d.fingerprintMu.RLock()
	defer d.fingerprintMu.RUnlock()

	requests, exists := d.fingerprints[fingerprint]
	if !exists {
		return 0
	}

	cutoff := time.Now().Add(-d.timeWindow)
	count := 0

	for _, t := range requests {
		if t.After(cutoff) {
			count++
		}
	}

	return count
}

// isSuspiciousIP checks if an IP is suspicious
func (d *Detector) isSuspiciousIP(ipAddress string) bool {
	ip := net.ParseIP(ipAddress)
	if ip == nil {
		return true // Invalid IP is suspicious
	}

	// Check if it's a known VPN/proxy range (simplified check)
	// In production, use a proper VPN/proxy detection service

	// Check for common cloud provider IPs
	if d.isCloudProviderIP(ipAddress) {
		return true
	}

	return false
}

// isCloudProviderIP checks if IP belongs to common cloud providers
func (d *Detector) isCloudProviderIP(ipAddress string) bool {
	// In production, check against actual cloud provider IP ranges
	// This is a simplified example

	// Common cloud provider IP ranges (very simplified)
	cloudRanges := []string{
		"3.0.0.0/8",     // AWS partial
		"13.0.0.0/8",    // AWS partial
		"35.0.0.0/8",    // GCP partial
		"104.0.0.0/8",   // Azure partial
	}

	ip := net.ParseIP(ipAddress)
	if ip == nil {
		return false
	}

	for _, cidr := range cloudRanges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}

		if ipNet.Contains(ip) {
			return true
		}
	}

	return false
}

// isBotLikeBehavior detects bot-like behavior patterns
func (d *Detector) isBotLikeBehavior(ipAddress string) bool {
	// Check if requests are coming too quickly
	d.ipMu.RLock()
	requests, exists := d.ipRequests[ipAddress]
	d.ipMu.RUnlock()

	if !exists || len(requests) < 2 {
		return false
	}

	// Check if requests are suspiciously regular (within 1 second intervals)
	recentRequests := make([]time.Time, 0)
	cutoff := time.Now().Add(-1 * time.Minute)

	for _, t := range requests {
		if t.After(cutoff) {
			recentRequests = append(recentRequests, t)
		}
	}

	if len(recentRequests) >= 3 {
		// Check if intervals are too regular
		intervals := make([]time.Duration, 0)
		for i := 1; i < len(recentRequests); i++ {
			interval := recentRequests[i].Sub(recentRequests[i-1])
			intervals = append(intervals, interval)
		}

		// If all intervals are within 2 seconds, it's suspicious
		allClose := true
		for _, interval := range intervals {
			if interval < time.Second || interval > 3*time.Second {
				allClose = false
				break
			}
		}

		if allClose {
			return true
		}
	}

	return false
}

// cleanup periodically cleans up old data
func (d *Detector) cleanup() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		cutoff := now.Add(-d.timeWindow)

		// Cleanup IP requests
		d.ipMu.Lock()
		for ip, requests := range d.ipRequests {
			newRequests := make([]time.Time, 0)
			for _, t := range requests {
				if t.After(cutoff) {
					newRequests = append(newRequests, t)
				}
			}

			if len(newRequests) > 0 {
				d.ipRequests[ip] = newRequests
			} else {
				delete(d.ipRequests, ip)
			}
		}
		d.ipMu.Unlock()

		// Cleanup address requests
		d.addressMu.Lock()
		for addr, requests := range d.addressRequests {
			newRequests := make([]time.Time, 0)
			for _, t := range requests {
				if t.After(cutoff) {
					newRequests = append(newRequests, t)
				}
			}

			if len(newRequests) > 0 {
				d.addressRequests[addr] = newRequests
			} else {
				delete(d.addressRequests, addr)
			}
		}
		d.addressMu.Unlock()

		// Cleanup fingerprints
		d.fingerprintMu.Lock()
		for fp, requests := range d.fingerprints {
			newRequests := make([]time.Time, 0)
			for _, t := range requests {
				if t.After(cutoff) {
					newRequests = append(newRequests, t)
				}
			}

			if len(newRequests) > 0 {
				d.fingerprints[fp] = newRequests
			} else {
				delete(d.fingerprints, fp)
			}
		}
		d.fingerprintMu.Unlock()

		// Cleanup blacklists
		d.blacklistMu.Lock()
		for ip, expiryTime := range d.blacklistedIPs {
			if now.After(expiryTime) {
				delete(d.blacklistedIPs, ip)
			}
		}
		for addr, expiryTime := range d.blacklistedAddresses {
			if now.After(expiryTime) {
				delete(d.blacklistedAddresses, addr)
			}
		}
		d.blacklistMu.Unlock()

		log.Println("Fraud detector cleanup completed")
	}
}

// GetStats returns fraud detection statistics
func (d *Detector) GetStats() map[string]interface{} {
	d.ipMu.RLock()
	ipCount := len(d.ipRequests)
	d.ipMu.RUnlock()

	d.addressMu.RLock()
	addressCount := len(d.addressRequests)
	d.addressMu.RUnlock()

	d.blacklistMu.RLock()
	blacklistedIPCount := len(d.blacklistedIPs)
	blacklistedAddressCount := len(d.blacklistedAddresses)
	d.blacklistMu.RUnlock()

	return map[string]interface{}{
		"tracked_ips":            ipCount,
		"tracked_addresses":      addressCount,
		"blacklisted_ips":        blacklistedIPCount,
		"blacklisted_addresses":  blacklistedAddressCount,
		"max_requests_per_ip":    d.maxRequestsPerIP,
		"max_requests_per_address": d.maxRequestsPerAddress,
		"time_window":            d.timeWindow.String(),
	}
}
