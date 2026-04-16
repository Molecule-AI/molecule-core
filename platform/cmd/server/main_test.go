package main

import (
	"os"
	"path/filepath"
	"testing"
)

// loadGitHubAppPrivateKey is the only platform-side wiring for the
// GitHub App private key. Two paths: file (preferred) and env-var
// (fallback). These tests pin the priority + the silent-fallback on
// missing-file behaviour.

func TestLoadGitHubAppPrivateKey_PrefersFile(t *testing.T) {
	tmp := t.TempDir()
	keyPath := filepath.Join(tmp, "key.pem")
	want := "-----BEGIN RSA PRIVATE KEY-----\nfile-contents\n-----END RSA PRIVATE KEY-----\n"
	if err := os.WriteFile(keyPath, []byte(want), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_APP_PRIVATE_KEY_FILE", keyPath)
	t.Setenv("GITHUB_APP_PRIVATE_KEY", "env-var-contents")

	got := loadGitHubAppPrivateKey()
	if string(got) != want {
		t.Errorf("file should win over env: got %q, want %q", string(got), want)
	}
}

func TestLoadGitHubAppPrivateKey_FallsBackToEnvWhenFileEmpty(t *testing.T) {
	t.Setenv("GITHUB_APP_PRIVATE_KEY_FILE", "")
	t.Setenv("GITHUB_APP_PRIVATE_KEY", "literal-pem")

	got := loadGitHubAppPrivateKey()
	if string(got) != "literal-pem" {
		t.Errorf("env should be used when file path empty: got %q", string(got))
	}
}

func TestLoadGitHubAppPrivateKey_FallsBackToEnvOnReadError(t *testing.T) {
	// File path set but file doesn't exist — should silently fall through
	// to env var. Operationally a missing file ≡ "not configured" rather
	// than "config error" — caller treats empty result as not-configured.
	t.Setenv("GITHUB_APP_PRIVATE_KEY_FILE", "/nonexistent/path/key.pem")
	t.Setenv("GITHUB_APP_PRIVATE_KEY", "fallback-pem")

	got := loadGitHubAppPrivateKey()
	if string(got) != "fallback-pem" {
		t.Errorf("missing file should fall back to env: got %q", string(got))
	}
}

func TestLoadGitHubAppPrivateKey_BothEmpty(t *testing.T) {
	t.Setenv("GITHUB_APP_PRIVATE_KEY_FILE", "")
	t.Setenv("GITHUB_APP_PRIVATE_KEY", "")

	got := loadGitHubAppPrivateKey()
	if len(got) != 0 {
		t.Errorf("both empty should return empty; got %d bytes", len(got))
	}
}
