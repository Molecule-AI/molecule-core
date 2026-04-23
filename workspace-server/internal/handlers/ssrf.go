package handlers

import (
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strings"
)

// ssrfCheckEnabled is the runtime gate for isSafeURL. Production keeps it
// true; tests that need to exercise loopback URLs (sqlmock + httptest
// servers bind to 127.0.0.1) flip it to false via setSSRFCheckForTest.
// Not safe with t.Parallel — tests that use the bypass must run serially.
var ssrfCheckEnabled = true

// setSSRFCheckForTest toggles the SSRF gate for the duration of a test.
// Only callable from this package's _test.go files (lowercase identifier).
func setSSRFCheckForTest(enabled bool) {
	ssrfCheckEnabled = enabled
}

// isSafeURL validates that a URL resolves to a publicly-routable address,
// preventing A2A requests from being redirected to internal/cloud-metadata
// infrastructure (SSRF, CWE-918). Workspace URLs come from DB/Redis caches
// so we validate before making any outbound HTTP call.
//
// SaaS relaxation: when saasMode() is true, RFC-1918 private ranges and
// IPv6 ULA are considered safe because workspaces live on sibling EC2s in
// the same VPC and register by their VPC-private IP. Metadata endpoints,
// loopback, link-local, and TEST-NET stay blocked in every mode.
func isSafeURL(rawURL string) error {
	if !ssrfCheckEnabled {
		return nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("forbidden scheme: %s (only http/https allowed)", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("empty hostname")
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() {
			return fmt.Errorf("forbidden loopback/unspecified/link-local IP: %s", ip)
		}
		if isPrivateOrMetadataIP(ip) {
			return fmt.Errorf("forbidden private/metadata IP: %s", ip)
		}
		return nil
	}
	addrs, err := net.LookupHost(host)
	if err != nil {
		return fmt.Errorf("DNS resolution blocked for hostname: %s (%v)", host, err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("DNS returned no addresses for: %s", host)
	}
	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip == nil {
			continue
		}
		if ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() {
			return fmt.Errorf("hostname %s resolves to forbidden link-local/loopback IP: %s", host, ip)
		}
		if isPrivateOrMetadataIP(ip) {
			return fmt.Errorf("hostname %s resolves to forbidden IP: %s", host, ip)
		}
	}
	return nil
}

// isPrivateOrMetadataIP returns true for IPs that must not be reached via A2A.
//
// Always blocked (both modes):
//   - 169.254.0.0/16 link-local (cloud metadata endpoints)
//   - 192.0.2.0/24, 198.51.100.0/24, 203.0.113.0/24 (TEST-NET RFC-5737)
//   - 100.64.0.0/10 (carrier-grade NAT)
//   - IPv6 loopback ::1, link-local fe80::/10, and ULA fc00::/7 in strict mode
//
// Allowed in SaaS mode only (saasMode() == true):
//   - 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16 (RFC-1918)
//   - fd00::/8 (IPv6 ULA subset of fc00::/7)
//
// Rationale: SaaS tenants run workspaces on sibling EC2s in the same VPC
// and register them by VPC-private IP. The control plane provisions these
// instances, so intra-VPC routing is trusted. On self-hosted / single-
// container deployments the relaxation is off and every private range
// stays blocked.
func isPrivateOrMetadataIP(ip net.IP) bool {
	saas := saasMode()

	// IPv4 path.
	if ip4 := ip.To4(); ip4 != nil {
		// Metadata link-local — always blocked.
		if metadataV4.Contains(ip4) {
			return true
		}
		// TEST-NET / documentation — always blocked.
		for _, r := range docRangesV4 {
			if r.Contains(ip4) {
				return true
			}
		}
		// Carrier-grade NAT — always blocked.
		if cgnatV4.Contains(ip4) {
			return true
		}
		// RFC-1918 private — blocked strict, allowed in SaaS.
		for _, r := range privateV4 {
			if r.Contains(ip4) {
				return !saas
			}
		}
		return false
	}

	// IPv6 path — .To4() was nil so this is a real v6 address.
	// ::1 (loopback) — treat as blocked here too for defense-in-depth.
	if ip.IsLoopback() {
		return true
	}
	// Link-local fe80::/10 — always blocked.
	if ip.IsLinkLocalUnicast() {
		return true
	}
	// ULA fc00::/7. fd00::/8 is the "locally assigned" half AWS hands out;
	// fc00::/8 is reserved. We treat the whole fc00::/7 as private, then
	// let SaaS relax fd00::/8 (matches the tests).
	if ulaV6.Contains(ip) {
		if saas && fd00V6.Contains(ip) {
			return false
		}
		return true
	}
	return false
}

var (
	metadataV4 = mustCIDR("169.254.0.0/16")
	cgnatV4    = mustCIDR("100.64.0.0/10")

	privateV4 = []net.IPNet{
		mustCIDR("10.0.0.0/8"),
		mustCIDR("172.16.0.0/12"),
		mustCIDR("192.168.0.0/16"),
	}
	docRangesV4 = []net.IPNet{
		mustCIDR("192.0.2.0/24"),
		mustCIDR("198.51.100.0/24"),
		mustCIDR("203.0.113.0/24"),
	}

	ulaV6  = mustCIDR("fc00::/7")
	fd00V6 = mustCIDR("fd00::/8")
)

func mustCIDR(s string) net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		panic("ssrf: bad CIDR " + s + ": " + err.Error())
	}
	return *n
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
