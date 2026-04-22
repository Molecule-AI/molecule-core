package handlers

import (
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strings"
)

// safeURLChecker is the active URL validator. Production code calls this
// exclusively — it delegates to isSafeURLDefault below. Tests swap in a
// permissive stub via setSSRFChecker (see ssrf_test.go).
//
// The var pattern avoids any sync.Once-per-package initialization issues
// and works cleanly with t.Cleanup in individual test cases.
var safeURLChecker = isSafeURLDefault

// setSSRFChecker is called exclusively by test helpers; not for production use.
var setSSRFChecker func(func(string) error) // nil in normal builds

// isSafeURL validates that a URL resolves to a publicly-routable address,
// preventing A2A requests from being redirected to internal/cloud-metadata
// infrastructure (SSRF, CWE-918). Workspace URLs come from DB/Redis caches
// so we validate before making any outbound HTTP call.
func isSafeURL(rawURL string) error { return safeURLChecker(rawURL) }

// isSafeURLDefault is the production URL validator.
func isSafeURLDefault(rawURL string) error {
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
	// Reject absolute paths (Unix and Windows) and ".." traversal BEFORE cleaning,
	// because filepath.Clean normalises ".." away before we can detect it.
	if filepath.IsAbs(filePath) || strings.HasPrefix(filePath, "..") || strings.Contains(filePath, "/..") || strings.Contains(filePath, "\\..") {
		return fmt.Errorf("path traversal or absolute path not allowed: %s", filePath)
	}
	// Also reject Windows absolute paths (e.g. C:\Windows\System32) on all platforms.
	// filepath.IsAbs only catches Unix-style (/...) on Linux, so we explicitly
	// check for drive-letter patterns.
	clean := filepath.Clean(filePath)
	if len(clean) >= 3 && clean[1] == ':' && (clean[2] == '\\' || clean[2] == '/') {
		return fmt.Errorf("path traversal or absolute path not allowed: %s", filePath)
	}
	return nil
}