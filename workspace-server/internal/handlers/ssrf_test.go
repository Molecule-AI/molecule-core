package handlers

import (
	"net"
	"testing"
)

// isSafeURL is defined in mcp.go.
// isPrivateOrMetadataIP is defined in mcp.go.

func TestIsPrivateOrMetadataIP(t *testing.T) {
	cases := []struct {
		name  string
		ipStr string
		want  bool
	}{
		// Must be blocked: RFC-1918 private
		{"10.0.0.1", "10.0.0.1", true},
		{"10.255.255.254", "10.255.255.254", true},
		{"172.16.0.0", "172.16.0.0", true},
		{"172.31.255.255", "172.31.255.255", true},
		{"192.168.0.1", "192.168.0.1", true},
		{"192.168.255.255", "192.168.255.255", true},
		// Must be blocked: cloud metadata link-local
		{"169.254.169.254", "169.254.169.254", true},
		{"169.254.0.1", "169.254.0.1", true},
		// Must be blocked: carrier-grade NAT
		{"100.64.0.1", "100.64.0.1", true},
		{"100.127.255.254", "100.127.255.254", true},
		// Must be blocked: documentation ranges
		{"192.0.2.1", "192.0.2.1", true},
		{"198.51.100.1", "198.51.100.1", true},
		{"203.0.113.1", "203.0.113.1", true},
		// Must be allowed: public IP addresses
		{"8.8.8.8", "8.8.8.8", false},
		{"1.1.1.1", "1.1.1.1", false},
		{"203.0.113.254", "203.0.113.254", false}, // TEST-NET-3 max — above 203.0.113.0/24 range end
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ip := net.ParseIP(tc.ipStr)
			if ip == nil {
				t.Fatalf("ParseIP(%q) returned nil", tc.ipStr)
			}
			got := isPrivateOrMetadataIP(ip)
			if got != tc.want {
				t.Errorf("isPrivateOrMetadataIP(%s) = %v, want %v", tc.ipStr, got, tc.want)
			}
		})
	}
}

func TestIsSafeURL(t *testing.T) {
	cases := []struct {
		name    string
		rawURL  string
		wantErr bool
	}{
		// Valid: public HTTPS
		{"public https", "https://agent.example.com:8080/a2a", false},
		{"public http", "http://agent.example.com/a2a", false},
		// Loopback is blocked by isSafeURL even in dev — the orchestrator
		// controls access via WorkspaceAuth + CanCommunicate, not via this URL check.
		// Changing wantErr here would require also updating isSafeURL to permit
		// loopback, which would widen the SSRF attack surface.
		{"localhost blocked", "http://127.0.0.1:8000", true},
		{"localhost with path", "http://127.0.0.1:9000", true},

		// Forbidden: non-HTTP(S) scheme
		{"file scheme blocked", "file:///etc/passwd", true},
		{"ftp scheme blocked", "ftp://internal/", true},
		{"mailto scheme blocked", "mailto://user@example.com", true},
		{"data scheme blocked", "data:text/html,<script>alert(1)</script>", true},

		// Forbidden: IP literals — cloud metadata
		{"AWS IMDS blocked", "http://169.254.169.254/latest/meta-data/", true},
		{"IMDS 169.254.0.1 blocked", "http://169.254.0.1/", true},

		// Forbidden: IP literals — loopback
		{"loopback 127.0.0.1 blocked", "http://127.0.0.1:8080", true},
		{"loopback 127.255.255.255 blocked", "http://127.255.255.255:9000", true},

		// Forbidden: IP literals — RFC-1918 private
		{"10.x private blocked", "http://10.0.0.1:8080", true},
		{"172.x private blocked", "http://172.16.0.5:8000", true},
		{"192.x private blocked", "http://192.168.1.1:8000", true},

		// Forbidden: IP literals — link-local multicast
		{"link-local multicast 224.0.0.1 blocked", "http://224.0.0.1/", true},
		{"link-local multicast 224.x.x.x blocked", "http://224.0.0.251:8080", true},

		// Forbidden: empty hostname
		{"empty hostname rejected", "http://:8080/a2a", true},

		// Forbidden: IP literals — unspecified
		{"0.0.0.0 blocked", "http://0.0.0.0:8080", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := isSafeURL(tc.rawURL)
			if tc.wantErr && err == nil {
				t.Errorf("isSafeURL(%q): expected error, got nil", tc.rawURL)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("isSafeURL(%q): expected nil, got %v", tc.rawURL, err)
			}
		})
	}
}