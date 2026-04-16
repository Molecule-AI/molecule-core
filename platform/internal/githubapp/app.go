// Package githubapp mints short-lived GitHub App installation tokens so
// every workspace container can authenticate to github.com under the
// Molecule AI bot identity instead of the CEO's personal PAT.
//
// Flow per workspace provision:
//  1. Platform signs an App JWT (RS256, 10-minute claim)
//  2. Platform POSTs to /app/installations/<id>/access_tokens — returns
//     a ~60-minute installation token scoped to the App's permissions
//  3. Platform passes that token to the workspace container as
//     GITHUB_TOKEN, overriding the previously-shared static PAT
//
// The JWT is never seen by the workspace. The installation token is
// short-lived, so if it leaks (container compromised, log file
// captured), damage is bounded to ~1 hour until rotation.
//
// Feature-flag behaviour: if any of the three required values
// (AppID, PrivateKeyPEM, InstallationID) is missing in platform config,
// NewClient returns nil and the provisioner falls back to the legacy
// GITHUB_TOKEN workspace secret path. That lets this PR ship without
// blocking on the GitHub-UI setup — operator enables by populating
// the three secrets via /admin/secrets after creating the App.
package githubapp

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenTTLBuffer is how long before actual expiry we consider a cached
// installation token "stale" and refresh it. GitHub issues tokens for
// ~60 minutes; we refresh at T-5 so a request landing at T-1 doesn't
// race against expiry mid-flight.
const TokenTTLBuffer = 5 * time.Minute

// JWTClaimTTL is the App JWT lifetime. GitHub caps this at 10 minutes;
// we use 9 to leave headroom for clock skew between our server and
// GitHub's verification.
const JWTClaimTTL = 9 * time.Minute

// GitHubAPIBase is the installation-tokens endpoint root. Overridden in
// tests to point at a local httptest server.
var GitHubAPIBase = "https://api.github.com"

// Config is the minimum input NewClient needs. All three fields must be
// non-zero; zero values disable the App auth path (fall back to static
// PAT, logged once on startup).
type Config struct {
	// AppID is the numeric GitHub App ID from the App's settings page.
	AppID int64
	// PrivateKeyPEM is the PEM-encoded RSA private key GitHub generated
	// when the App was created. Must be the full `-----BEGIN RSA PRIVATE
	// KEY-----` block including headers.
	PrivateKeyPEM []byte
	// InstallationID is the numeric installation ID for the org/account
	// where the App is installed (visible at
	// https://github.com/organizations/<org>/settings/installations/<id>).
	// Per-repo installs work too; the ID is still the "installation" ID.
	InstallationID int64
	// HTTPClient is optional; defaults to http.DefaultClient with a
	// 30s timeout.
	HTTPClient *http.Client
}

// Client caches installation tokens and refreshes them on demand. Safe
// for concurrent use.
type Client struct {
	cfg        Config
	privateKey *rsa.PrivateKey
	httpClient *http.Client

	mu          sync.Mutex
	cachedToken string
	expiresAt   time.Time
}

// NewClient parses the private key and returns a ready-to-use client.
// Returns (nil, nil) if any required field is empty — the caller
// interprets this as "App auth not configured; fall back to PAT".
// Returns (nil, err) only when config is provided but malformed
// (bad PEM, non-RSA key, etc.) — that's a hard configuration bug.
func NewClient(cfg Config) (*Client, error) {
	if cfg.AppID == 0 || cfg.InstallationID == 0 || len(cfg.PrivateKeyPEM) == 0 {
		return nil, nil
	}
	block, _ := pem.Decode(cfg.PrivateKeyPEM)
	if block == nil {
		return nil, errors.New("githubapp: private key is not valid PEM")
	}
	// GitHub's key is PKCS#1 ("RSA PRIVATE KEY") by default, but some
	// downloads are PKCS#8 ("PRIVATE KEY") — accept both.
	var privateKey *rsa.PrivateKey
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		privateKey = k
	} else {
		parsed, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("githubapp: parse private key: PKCS1=%v PKCS8=%v", err, err2)
		}
		rsaKey, ok := parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("githubapp: private key is not RSA (got %T)", parsed)
		}
		privateKey = rsaKey
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{
		cfg:        cfg,
		privateKey: privateKey,
		httpClient: httpClient,
	}, nil
}

// InstallationToken returns a currently-valid installation token,
// minting a fresh one when the cache is empty or the existing token is
// within TokenTTLBuffer of expiry. Safe for concurrent callers: one
// mint per expiry window; the rest share the result.
func (c *Client) InstallationToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cachedToken != "" && time.Until(c.expiresAt) > TokenTTLBuffer {
		return c.cachedToken, nil
	}

	jwt, err := c.signAppJWT(time.Now())
	if err != nil {
		return "", fmt.Errorf("githubapp: sign JWT: %w", err)
	}
	token, expiresAt, err := c.fetchInstallationToken(ctx, jwt)
	if err != nil {
		return "", err
	}
	c.cachedToken = token
	c.expiresAt = expiresAt
	return token, nil
}

// signAppJWT produces the RS256 JWT GitHub requires for App-auth
// endpoints. Extracted for test coverage.
func (c *Client) signAppJWT(now time.Time) (string, error) {
	claims := jwt.MapClaims{
		"iat": now.Add(-30 * time.Second).Unix(), // backdate 30s for clock skew
		"exp": now.Add(JWTClaimTTL).Unix(),
		"iss": c.cfg.AppID,
	}
	t := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return t.SignedString(c.privateKey)
}

// fetchInstallationToken calls GitHub's installation-token endpoint.
// Returns the token, its expiry, or an error describing the failure
// with the response body truncated (so logs don't grow unbounded if
// GitHub starts returning HTML under outage).
func (c *Client) fetchInstallationToken(ctx context.Context, appJWT string) (string, time.Time, error) {
	url := fmt.Sprintf("%s/app/installations/%d/access_tokens", GitHubAPIBase, c.cfg.InstallationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("githubapp: new request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+appJWT)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("githubapp: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", time.Time{}, fmt.Errorf("githubapp: installation token fetch returned HTTP %d: %s", resp.StatusCode, string(body))
	}
	var out struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", time.Time{}, fmt.Errorf("githubapp: decode response: %w", err)
	}
	if out.Token == "" || out.ExpiresAt.IsZero() {
		return "", time.Time{}, errors.New("githubapp: empty token or expiry in response")
	}
	return out.Token, out.ExpiresAt, nil
}
