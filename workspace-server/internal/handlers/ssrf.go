package handlers

import (
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strings"
)

// saasMode is defined in registry.go and returns true when the platform is
// running in SaaS multi-tenant mode (vs self-hosted single-tenant).

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

// isPrivateOrMetadataIP returns true for cloud-metadata / loopback / link-local
// ranges (always) and RFC-1918 / IPv6 ULA ranges (self-hosted only).
//
// In SaaS cross-EC2 mode (see saasMode() in registry.go) the tenant platform
// and its workspaces share a VPC, so workspaces register with their
// VPC-private IP — typically 172.31.x.x on AWS default VPCs. Blocking RFC-1918
// unconditionally would reject every legitimate registration. Cloud metadata
// (169.254.0.0/16, fe80::/10), loopback, and TEST-NET ranges stay blocked in
// both modes; they are never a legitimate agent URL.
//
// Both IPv4 and IPv6 are checked. The previous implementation returned false
// for every non-IPv4 input, which meant a registered [::1] or [fe80::…]
// URL would bypass the SSRF gate entirely.
func isPrivateOrMetadataIP(ip net.IP) bool {
	// Always blocked — IPv4 cloud metadata + network-test ranges.
	metadataRangesV4 := []string{
		"169.254.0.0/16",  // link-local / IMDSv1-v2
		"100.64.0.0/10",   // CGNAT — reachable via some VPC configs, not a legit agent URL
		"192.0.2.0/24",    // TEST-NET-1
		"198.51.100.0/24", // TEST-NET-2
		"203.0.113.0/24",  // TEST-NET-3
	}
	// Always blocked — IPv6 cloud-metadata / loopback equivalents.
	metadataRangesV6 := []string{
		"::1/128",       // loopback
		"fe80::/10",     // link-local (IMDS analogue)
		"::ffff:0:0/96", // IPv4-mapped loopback (defence-in-depth; To4() below usually normalises first)
	}
	// RFC-1918 private — blocked in self-hosted, allowed in SaaS.
	rfc1918RangesV4 := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}
	// RFC-4193 ULA — IPv6 analogue of RFC-1918. Same SaaS-mode treatment.
	ulaRangesV6 := []string{
		"fc00::/7",
	}

	contains := func(cidrs []string, target net.IP) bool {
		for _, c := range cidrs {
			_, n, err := net.ParseCIDR(c)
			if err != nil {
				continue
			}
			if n.Contains(target) {
				return true
			}
		}
		return false
	}

	if ip4 := ip.To4(); ip4 != nil {
		if contains(metadataRangesV4, ip4) {
			return true
		}
		if saasMode() {
			return false
		}
		return contains(rfc1918RangesV4, ip4)
	}

	if contains(metadataRangesV6, ip) {
		return true
	}
	if saasMode() {
		return false
	}
	return contains(ulaRangesV6, ip)
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