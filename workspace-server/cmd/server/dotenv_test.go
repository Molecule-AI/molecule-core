package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDotEnvLine(t *testing.T) {
	cases := []struct {
		in      string
		k, v    string
		ok      bool
		comment string
	}{
		{in: "", ok: false, comment: "empty line"},
		{in: "   ", ok: false, comment: "whitespace-only"},
		{in: "# top-level comment", ok: false, comment: "full-line comment"},
		{in: "  #  indented comment", ok: false, comment: "indented full-line comment"},
		{in: "FOO", ok: false, comment: "no equals"},
		{in: "=BAR", ok: false, comment: "missing key"},

		{in: "FOO=bar", k: "FOO", v: "bar", ok: true, comment: "plain"},
		{in: "  FOO=bar", k: "FOO", v: "bar", ok: true, comment: "leading whitespace"},
		{in: "FOO=bar   ", k: "FOO", v: "bar", ok: true, comment: "trailing whitespace stripped"},
		{in: "FOO  =bar", k: "FOO", v: "bar", ok: true, comment: "whitespace before equals"},

		{in: "FOO=bar # comment", k: "FOO", v: "bar", ok: true, comment: "inline space-hash comment"},
		{in: "FOO=bar\t# comment", k: "FOO", v: "bar", ok: true, comment: "inline tab-hash comment"},
		{in: "FOO=bar    # lots of spaces", k: "FOO", v: "bar", ok: true, comment: "multiple spaces before hash"},

		{in: "FOO=bar#nocomment", k: "FOO", v: "bar#nocomment", ok: true, comment: "bare hash inside value preserved"},
		{in: "URL=postgres://u:p@h:5432/db?sslmode=disable", k: "URL", v: "postgres://u:p@h:5432/db?sslmode=disable", ok: true, comment: "url with embedded equals"},
		{in: "TOKEN=eyJhbGciOiJIUzI1NiJ9.payload.sig=", k: "TOKEN", v: "eyJhbGciOiJIUzI1NiJ9.payload.sig=", ok: true, comment: "base64 padding preserved"},

		{in: "FOO=", k: "FOO", v: "", ok: true, comment: "empty value"},
		{in: "ADMIN_TOKEN=", k: "ADMIN_TOKEN", v: "", ok: true, comment: "empty value (production gate sentinel)"},
	}

	for _, tc := range cases {
		t.Run(tc.comment, func(t *testing.T) {
			k, v, ok := parseDotEnvLine(tc.in)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v (input=%q)", ok, tc.ok, tc.in)
			}
			if !tc.ok {
				return
			}
			if k != tc.k || v != tc.v {
				t.Fatalf("got (%q, %q), want (%q, %q)", k, v, tc.k, tc.v)
			}
		})
	}
}

func TestLoadDotEnvIfPresent_PreservesExisting(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	body := []byte("DOTENV_TEST_NEW=from_file\nDOTENV_TEST_EXISTING=from_file\n")
	if err := os.WriteFile(envPath, body, 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	// Pre-set one of the keys — file value must NOT clobber it.
	t.Setenv("DOTENV_TEST_EXISTING", "from_real_env")
	// Ensure the other key starts unset.
	os.Unsetenv("DOTENV_TEST_NEW")
	t.Cleanup(func() { os.Unsetenv("DOTENV_TEST_NEW") })

	// Run from the temp dir so findDotEnv picks our fixture.
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	loadDotEnvIfPresent()

	if got := os.Getenv("DOTENV_TEST_NEW"); got != "from_file" {
		t.Errorf("DOTENV_TEST_NEW = %q, want %q", got, "from_file")
	}
	if got := os.Getenv("DOTENV_TEST_EXISTING"); got != "from_real_env" {
		t.Errorf("existing env clobbered: got %q, want %q", got, "from_real_env")
	}
}

func TestLoadDotEnvIfPresent_NoFile_NoOp(t *testing.T) {
	dir := t.TempDir() // empty — no .env at this level
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	// Should not panic, log loud errors, or set anything. Best-effort
	// silent miss is the contract.
	loadDotEnvIfPresent()
}

func TestFindDotEnv_WalksUpward(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("X=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	got, ok := findDotEnv()
	if !ok {
		t.Fatal("expected to find .env walking upward")
	}
	want := filepath.Join(root, ".env")
	// macOS resolves /var → /private/var on TempDir, so compare via
	// EvalSymlinks for both sides to dodge that.
	gotR, _ := filepath.EvalSymlinks(got)
	wantR, _ := filepath.EvalSymlinks(want)
	if gotR != wantR {
		t.Errorf("findDotEnv() = %q, want %q", got, want)
	}
}
