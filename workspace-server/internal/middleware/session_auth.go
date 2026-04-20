package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// sessionCache holds short-lived positive results for upstream-verified
// session cookies. Keyed by the raw Cookie header value so ANY change
// (logout, fresh session) invalidates by just being different bytes.
//
// TTL is deliberately short — 30s — because the SaaS session lives on
// the CP; if ops revokes a token, we want that reflected quickly. A
// longer TTL would let revoked sessions drift into the tenant. 30s is
// the sweet spot: fast enough for security, slow enough to avoid CP
// hammering on every canvas render.
var sessionCache sync.Map

const sessionCacheTTL = 30 * time.Second

type sessionCacheEntry struct {
	verifiedAt time.Time
	ok         bool
}

// cpSessionEndpointURL is where we verify. Reads the same env the
// router uses for the /cp/* reverse-proxy. Empty string → feature
// disabled (self-hosted / dev). Computed at first call so tests can
// override via env.
func cpSessionEndpointURL() string {
	base := strings.TrimRight(os.Getenv("CP_UPSTREAM_URL"), "/")
	if base == "" {
		return ""
	}
	return base + "/cp/auth/me"
}

// verifiedCPSession returns true when the request carries a cookie
// that the CP recognizes as a logged-in user. Caches positive results
// for sessionCacheTTL so burst canvas renders don't fan out to the CP
// on every admin fetch.
//
// Returns (false, false) when there is no cookie at all — callers
// distinguish "no credential presented" (fall through to other tiers)
// from "credential presented but invalid" (abort with 401).
func verifiedCPSession(cookieHeader string) (valid, presented bool) {
	if cookieHeader == "" {
		return false, false
	}
	endpoint := cpSessionEndpointURL()
	if endpoint == "" {
		return false, true
	}

	// Cache lookup.
	if v, ok := sessionCache.Load(cookieHeader); ok {
		e := v.(sessionCacheEntry)
		if time.Since(e.verifiedAt) < sessionCacheTTL {
			return e.ok, true
		}
		sessionCache.Delete(cookieHeader)
	}

	// Fetch /cp/auth/me with the presented cookie. Short timeout —
	// a slow CP mustn't gate every canvas page render.
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		log.Printf("verifiedCPSession: build req: %v", err)
		return false, true
	}
	req.Header.Set("Cookie", cookieHeader)
	// Browser-style User-Agent so the CP's bot-detection (if any)
	// doesn't block us; we're a legitimate proxy for the UI.
	req.Header.Set("User-Agent", "molecule-tenant-platform/session-verifier")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("verifiedCPSession: upstream: %v", err)
		return false, true
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		sessionCache.Store(cookieHeader, sessionCacheEntry{verifiedAt: time.Now(), ok: false})
		return false, true
	}

	// Parse minimally to make sure it's actually a session object, not
	// an HTML error page from an upstream proxy shell.
	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil || body.UserID == "" {
		sessionCache.Store(cookieHeader, sessionCacheEntry{verifiedAt: time.Now(), ok: false})
		return false, true
	}

	sessionCache.Store(cookieHeader, sessionCacheEntry{verifiedAt: time.Now(), ok: true})
	return true, true
}
