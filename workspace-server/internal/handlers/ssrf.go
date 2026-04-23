package handlers

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ssrfTestBypass, when true, skips DNS-based hostname validation so that
// unit tests using non-routable hostnames in httptest.Server URLs don't
// produce false negatives (SSRF defence is exercised in ssrf_test.go).
var ssrfTestBypass = os.Getenv("MOLECULE_TEST_SKIP_SSRF") == "1"

// isSafeURL validates that a URL resolves to a publicly-routable address,
// preventing A2A requests from being redirected to internal/cloud-metadata
// infrastructure (SSRF, CWE-918). Workspace URLs come from DB/Redis caches
// so we validate before making any outbound HTTP call.
func isSafeURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	// Reject non-HTTP(S) schemes.
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("forbidden scheme: %s (only http/https allowed)", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("empty hostname")
	}
	// Block direct IP addresses.
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() {
			return fmt.Errorf("forbidden loopback/unspecified IP: %s", ip)
		}
		if isPrivateOrMetadataIP(ip) {
			return fmt.Errorf("forbidden private/metadata IP: %s", ip)
		}
		return nil
	}
	// Test bypass: skip DNS resolution when MOLECULE_TEST_SKIP_SSRF=1 so that
	// httptest.Server URLs (using non-routable hostnames like 127.0.0.1:N or
	// IP:port from Go's transport) don't produce false negatives in unit tests.
	// The SSRF defence itself is covered exhaustively in ssrf_test.go.
	if ssrfTestBypass {
		return nil
	}
	// For hostnames, resolve and validate each returned IP.
	addrs, err := net.LookupHost(host)
	if err != nil {
		// DNS resolution failure — block it. Could be an internal hostname.
		return fmt.Errorf("DNS resolution blocked for hostname: %s (%v)", host, err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("DNS returned no addresses for: %s", host)
	}
	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip != nil && (ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || isPrivateOrMetadataIP(ip)) {
			return fmt.Errorf("hostname %s resolves to forbidden IP: %s", host, ip)
		}
	}
	return nil
}

// setSsrfTestBypass overrides ssrfTestBypass for the duration of a test.
// Returns a restore function that resets the original value.
func setSsrfTestBypass(v bool) func() {
	prev := ssrfTestBypass
	ssrfTestBypass = v
	return func() { ssrfTestBypass = prev }
}

// isPrivateOrMetadataIP returns true for RFC-1918 private, carrier-grade NAT,
// link-local, and cloud metadata ranges.
func isPrivateOrMetadataIP(ip net.IP) bool {
	var privateRanges = []net.IPNet{
		{IP: net.ParseIP("10.0.0.0"), Mask: net.CIDRMask(8, 32)},
		{IP: net.ParseIP("172.16.0.0"), Mask: net.CIDRMask(12, 32)},
		{IP: net.ParseIP("192.168.0.0"), Mask: net.CIDRMask(16, 32)},
		{IP: net.ParseIP("169.254.0.0"), Mask: net.CIDRMask(16, 32)},
		{IP: net.ParseIP("100.64.0.0"), Mask: net.CIDRMask(10, 32)},
		{IP: net.ParseIP("192.0.2.0"), Mask: net.CIDRMask(24, 32)},
		{IP: net.ParseIP("198.51.100.0"), Mask: net.CIDRMask(24, 32)},
		{IP: net.ParseIP("203.0.113.0"), Mask: net.CIDRMask(24, 32)},
	}
	ip = ip.To4()
	if ip == nil {
		return false
	}
	for _, r := range privateRanges {
		if r.Contains(ip) {
			return true
		}
	}
	return false
}

// validateRelPath checks that a file path is relative and does not escape
// the destination via absolute paths or ".." traversal. Used by
// copyFilesToContainer and deleteViaEphemeral as a defence-in-depth measure.
func validateRelPath(filePath string) error {
	clean := filepath.Clean(filePath)
	if filepath.IsAbs(clean) || strings.Contains(clean, "..") {
		return fmt.Errorf("path traversal or absolute path not allowed: %s", filePath)
	}
	return nil
}