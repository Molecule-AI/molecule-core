// Package wsauth — workspace authentication tokens (Phase 30.1).
//
// Tokens are opaque random strings (256 bits, base64url-encoded). The
// plaintext is returned to the agent exactly once at issuance time; only
// sha256(plaintext) is ever stored in the database. The agent presents the
// token on every subsequent request via the `Authorization: Bearer <token>`
// header. The ValidateToken function looks up the hash, confirms the
// workspace matches, updates last_used_at, and returns the workspace ID.
//
// This package deliberately avoids JWT — we don't need signed claims, only
// opaque bearer credentials that can be rotated and revoked per workspace.
package wsauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// tokenPayloadBytes controls the raw-random length of a token before
// base64-encoding. 32 bytes → 256-bit entropy → 43-char URL-safe string,
// which comfortably resists guessing attacks over the public internet.
const tokenPayloadBytes = 32

// tokenPrefixLen is how many leading characters we keep in the `prefix`
// column for display / debugging. Short enough to reveal nothing usable;
// long enough to correlate log lines with rotated tokens.
const tokenPrefixLen = 8

// ErrInvalidToken is returned by ValidateToken when the presented token
// doesn't match a live row. Callers should return HTTP 401 on this error —
// do NOT leak the underlying database error or whether the workspace ID
// was known.
var ErrInvalidToken = errors.New("invalid or revoked workspace token")

// IssueToken mints a fresh token, stores its hash + prefix against the
// given workspace, and returns the plaintext to show the caller exactly
// once. The plaintext is never recoverable from the database afterwards.
//
// Callers should treat the returned string as secret material and pass it
// straight to the agent (env var, bundle response body, etc.) without
// logging it.
func IssueToken(ctx context.Context, db *sql.DB, workspaceID string) (string, error) {
	buf := make([]byte, tokenPayloadBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("wsauth: generate token: %w", err)
	}
	plaintext := base64.RawURLEncoding.EncodeToString(buf)

	hash := sha256.Sum256([]byte(plaintext))
	prefix := plaintext[:tokenPrefixLen]

	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace_auth_tokens (workspace_id, token_hash, prefix)
		VALUES ($1, $2, $3)
	`, workspaceID, hash[:], prefix)
	if err != nil {
		return "", fmt.Errorf("wsauth: persist token: %w", err)
	}
	return plaintext, nil
}

// ValidateToken confirms the presented plaintext matches a live row whose
// workspace_id equals expectedWorkspaceID. On success it refreshes
// last_used_at (best-effort — failure to update is logged by the caller,
// not propagated as an auth failure).
//
// The expectedWorkspaceID binding is required because a token is only
// valid for the workspace it was issued to. A compromised token from
// workspace A must never authenticate workspace B.
func ValidateToken(ctx context.Context, db *sql.DB, expectedWorkspaceID, plaintext string) error {
	if plaintext == "" || expectedWorkspaceID == "" {
		return ErrInvalidToken
	}
	hash := sha256.Sum256([]byte(plaintext))

	var tokenID, workspaceID string
	err := db.QueryRowContext(ctx, `
		SELECT id, workspace_id
		FROM workspace_auth_tokens
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, hash[:]).Scan(&tokenID, &workspaceID)
	if err != nil {
		// Includes sql.ErrNoRows — collapse to a single public-facing error
		// so the handler can't accidentally leak which half of the check
		// failed (bad token vs. wrong workspace).
		return ErrInvalidToken
	}
	if workspaceID != expectedWorkspaceID {
		return ErrInvalidToken
	}

	// Best-effort last_used_at update. A failure here (DB hiccup, etc.)
	// must not cause an otherwise-valid request to 401.
	_, _ = db.ExecContext(ctx,
		`UPDATE workspace_auth_tokens SET last_used_at = now() WHERE id = $1`, tokenID)
	return nil
}

// RevokeAllForWorkspace invalidates every live token for a workspace.
// Called from the workspace-delete handler so compromised credentials
// can't outlive the workspace, and from future rotation flows.
func RevokeAllForWorkspace(ctx context.Context, db *sql.DB, workspaceID string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE workspace_auth_tokens
		SET revoked_at = now()
		WHERE workspace_id = $1 AND revoked_at IS NULL
	`, workspaceID)
	if err != nil {
		return fmt.Errorf("wsauth: revoke: %w", err)
	}
	return nil
}

// HasAnyLiveToken reports whether the given workspace has at least one
// live (non-revoked) token on file. Used by the lazy-bootstrap path in
// the heartbeat handler — a legacy workspace that registered before
// tokens existed needs exactly one issued on its first post-upgrade
// heartbeat rather than being rejected outright.
func HasAnyLiveToken(ctx context.Context, db *sql.DB, workspaceID string) (bool, error) {
	var n int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM workspace_auth_tokens
		WHERE workspace_id = $1 AND revoked_at IS NULL
	`, workspaceID).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// BearerTokenFromHeader extracts the token from an Authorization header
// value. Returns the empty string if the header is missing or malformed,
// which callers MUST treat as an authentication failure — we deliberately
// do not return an error so the handler control-flow stays `if token == ""`
// rather than `if err != nil`.
func BearerTokenFromHeader(h string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return ""
	}
	return strings.TrimSpace(h[len(prefix):])
}
