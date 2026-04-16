package githubapp

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// genTestKey produces a fresh RSA-2048 key, PEM-encoded as PKCS#1
// ("-----BEGIN RSA PRIVATE KEY-----" style) matching what GitHub emits
// on App creation. Returns the PEM bytes plus the parsed key so tests
// can verify JWT signatures locally.
func genTestKey(t *testing.T) ([]byte, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa gen: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	return pemBytes, key
}

func TestNewClient_DisabledWhenConfigMissing(t *testing.T) {
	// All three fields empty — NewClient returns (nil, nil) so the
	// caller treats it as "App auth not configured, fall back to PAT".
	c, err := NewClient(Config{})
	if err != nil {
		t.Fatalf("unexpected error for empty config: %v", err)
	}
	if c != nil {
		t.Errorf("expected nil client, got %v", c)
	}
	// One field missing = still disabled.
	pem, _ := genTestKey(t)
	c, err = NewClient(Config{AppID: 123, PrivateKeyPEM: pem})
	if err != nil {
		t.Fatalf("unexpected error for partial config: %v", err)
	}
	if c != nil {
		t.Errorf("expected nil client when installation ID missing")
	}
}

func TestNewClient_RejectsBadPEM(t *testing.T) {
	_, err := NewClient(Config{
		AppID:          123,
		InstallationID: 456,
		PrivateKeyPEM:  []byte("not a pem block"),
	})
	if err == nil {
		t.Error("expected error for non-PEM input")
	}
}

func TestNewClient_AcceptsPKCS1AndPKCS8(t *testing.T) {
	// PKCS#1 (what GitHub emits).
	pkcs1, _ := genTestKey(t)
	if _, err := NewClient(Config{AppID: 1, InstallationID: 1, PrivateKeyPEM: pkcs1}); err != nil {
		t.Errorf("PKCS#1 should parse: %v", err)
	}

	// PKCS#8 (some re-encoded downloads) — build one ourselves.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pkcs8Bytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	pkcs8 := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8Bytes})
	if _, err := NewClient(Config{AppID: 1, InstallationID: 1, PrivateKeyPEM: pkcs8}); err != nil {
		t.Errorf("PKCS#8 should parse: %v", err)
	}
}

func TestSignAppJWT_ClaimStructure(t *testing.T) {
	pem, key := genTestKey(t)
	c, err := NewClient(Config{AppID: 42, InstallationID: 99, PrivateKeyPEM: pem})
	if err != nil {
		t.Fatal(err)
	}

	// Use time.Now so jwt.Parse's default claim validation (checks exp
	// against wall-clock) doesn't flake as the fixed date drifts into
	// the past. Previous fixed time.Date(2026,4,16,...) caused the
	// signed token's 9-minute TTL to expire before parse on the next
	// rebase test run, giving a "token is expired" parse error that
	// had nothing to do with the code under test.
	now := time.Now()
	signed, err := c.signAppJWT(now)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	// Parse back with the public half, assert the claim shape matches
	// GitHub's requirements (iss = AppID, exp ≤ iat + 10 min).
	parsed, err := jwt.Parse(signed, func(_ *jwt.Token) (interface{}, error) {
		return &key.PublicKey, nil
	})
	if err != nil {
		t.Fatalf("parse signed token: %v", err)
	}
	claims := parsed.Claims.(jwt.MapClaims)
	if iss, _ := claims["iss"].(float64); int64(iss) != 42 {
		t.Errorf("iss: got %v, want 42", claims["iss"])
	}
	iat, _ := claims["iat"].(float64)
	exp, _ := claims["exp"].(float64)
	if exp-iat > 600 { // 10 min
		t.Errorf("token lifetime > 10 min (%v), GitHub will reject", exp-iat)
	}
	if exp <= iat {
		t.Errorf("exp (%v) not after iat (%v)", exp, iat)
	}
}

// mockGitHub stands in for api.github.com during tests. It expects
// POST /app/installations/{id}/access_tokens with a Bearer <JWT>, and
// returns a fake installation token + expiry.
func mockGitHub(t *testing.T, wantInstallID int64, token string, expiry time.Time, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantPath := fmt.Sprintf("/app/installations/%d/access_tokens", wantInstallID)
		if r.URL.Path != wantPath {
			http.Error(w, "wrong path: "+r.URL.Path, http.StatusNotFound)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "wrong method", http.StatusMethodNotAllowed)
			return
		}
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, "missing bearer", http.StatusUnauthorized)
			return
		}
		if status != http.StatusCreated {
			http.Error(w, "mock error", status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":      token,
			"expires_at": expiry.Format(time.RFC3339),
		})
	}))
}

func TestInstallationToken_MintsAndCaches(t *testing.T) {
	pem, _ := genTestKey(t)
	expiry := time.Now().Add(60 * time.Minute)
	srv := mockGitHub(t, 99, "ghs_fakeinstalltoken", expiry, http.StatusCreated)
	defer srv.Close()

	old := GitHubAPIBase
	GitHubAPIBase = srv.URL
	defer func() { GitHubAPIBase = old }()

	c, err := NewClient(Config{AppID: 42, InstallationID: 99, PrivateKeyPEM: pem})
	if err != nil {
		t.Fatal(err)
	}

	// First call — mints.
	tok1, err := c.InstallationToken(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if tok1 != "ghs_fakeinstalltoken" {
		t.Errorf("got %q, want ghs_fakeinstalltoken", tok1)
	}

	// Second call — cache hit. Shut down the mock to prove we don't
	// re-fetch; the cached value must still come back.
	srv.Close()
	tok2, err := c.InstallationToken(context.Background())
	if err != nil {
		t.Errorf("cache miss after fresh mint: %v", err)
	}
	if tok2 != tok1 {
		t.Errorf("cached token differs: %q vs %q", tok2, tok1)
	}
}

func TestInstallationToken_RefreshesNearExpiry(t *testing.T) {
	pem, _ := genTestKey(t)

	var mu sync.Mutex
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		call := calls
		mu.Unlock()
		w.WriteHeader(http.StatusCreated)
		// First call: expiry within the refresh buffer → cache should be
		// treated as stale on next request. Second call: fresh 60 min.
		var expiry time.Time
		var tok string
		if call == 1 {
			expiry = time.Now().Add(TokenTTLBuffer / 2) // already stale
			tok = "ghs_first"
		} else {
			expiry = time.Now().Add(60 * time.Minute)
			tok = "ghs_second"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":      tok,
			"expires_at": expiry.Format(time.RFC3339),
		})
	}))
	defer srv.Close()

	old := GitHubAPIBase
	GitHubAPIBase = srv.URL
	defer func() { GitHubAPIBase = old }()

	c, _ := NewClient(Config{AppID: 42, InstallationID: 99, PrivateKeyPEM: pem})

	t1, _ := c.InstallationToken(context.Background())
	if t1 != "ghs_first" {
		t.Errorf("first mint: got %q", t1)
	}
	// Second call finds the token inside the buffer → refetches.
	t2, _ := c.InstallationToken(context.Background())
	if t2 != "ghs_second" {
		t.Errorf("second mint: got %q", t2)
	}
	if calls != 2 {
		t.Errorf("expected 2 GitHub calls, got %d", calls)
	}
}

func TestInstallationToken_PropagatesGitHubError(t *testing.T) {
	pem, _ := genTestKey(t)
	srv := mockGitHub(t, 99, "", time.Time{}, http.StatusUnauthorized)
	defer srv.Close()

	old := GitHubAPIBase
	GitHubAPIBase = srv.URL
	defer func() { GitHubAPIBase = old }()

	c, _ := NewClient(Config{AppID: 42, InstallationID: 99, PrivateKeyPEM: pem})
	_, err := c.InstallationToken(context.Background())
	if err == nil {
		t.Fatal("expected error when GitHub returns 401")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should surface the HTTP status, got: %v", err)
	}
}
