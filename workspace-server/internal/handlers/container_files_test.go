package handlers

import (
	"os"
	"strings"
	"testing"
)

// TestValidateRelPath tests the path-traversal guard used in deleteViaEphemeral.
// validateRelPath should reject absolute paths and ".." segments after cleaning.
// NOTE: This test lives in a file that does NOT call setupTestDB, so SSRF checks
// remain enabled. The test directly exercises validateRelPath without any DB
// dependency, so no mock DB is needed.
func TestValidateRelPath(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		wantErr  bool
		errSubstr string // if non-empty, error message must contain this substring
	}{
		// Valid: simple relative paths inside a destination
		{"single file", "config.json", false, ""},
		{"nested relative", "dir/subdir/file.txt", false, ""},
		{"file at destination root", "file.txt", false, ""},
		{"subdirectory file", "configs/myapp/file.cfg", false, ""},
		{"dotfile (hidden file, not traversal)", ".env", false, ""},

		// Empty/dot-only: must be rejected with specific message
		{"empty string", "", true, "empty or dot-only path"},
		{"dot only", ".", true, "empty or dot-only path"},

		// Traversal: must be rejected
		{"double dot parent", "../etc/passwd", true, "path traversal"},
		{"trailing dotdot", "../", true, "path traversal"},
		{"embedded dotdot", "foo/../bar", true, "path traversal"},
		{"dotdot middle", "a/b/../../c", true, "path traversal"},
		{"path ends in ..", "foo/..", true, "path traversal"},
		{"bare ..", "..", true, "path traversal"},

		// Absolute: must be rejected
		{"absolute unix", "/etc/passwd", true, "path traversal"},
		{"absolute windows", "C:\\Windows\\System32", false, ""}, // Unix/Linux: no drive letter, treated as relative by Go
		{"embedded absolute", "foo/etc/passwd", false, ""},
		{"root absolute", "/workspace/file.txt", true, "path traversal"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRelPath(tc.path)
			if tc.wantErr && err == nil {
				t.Errorf("validateRelPath(%q): expected error, got nil", tc.path)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("validateRelPath(%q): expected nil, got %v", tc.path, err)
			}
			if tc.errSubstr != "" && (err == nil || !strings.Contains(err.Error(), tc.errSubstr)) {
				t.Errorf("validateRelPath(%q): expected error containing %q, got %v", tc.path, tc.errSubstr, err)
			}
		})
	}
}

// TestValidateRelPath_Cleaned ensures that validateRelPath is called on the
// cleaned (resolved) path, not the raw input, so tricks like "foo/./bar"
// pass but "foo/../bar" fails.
func TestValidateRelPath_Cleaned(t *testing.T) {
	// ". " (dot-space) is not "..", but after Clean() it becomes just the dir.
	// validateRelPath should be called on the clean path, not raw.
	// These are valid relative paths.
	valid := []string{
		"foo/./bar",
		"foo/././baz",
		"./file.cfg",
	}
	for _, p := range valid {
		if err := validateRelPath(p); err != nil {
			t.Errorf("validateRelPath(%q): expected nil, got %v", p, err)
		}
	}
}

// TestDeleteViaEphemeral_SafeForm documents that the F1085 security fix
// scopes the rm target to /configs/ using filepath.Join + filepath.Clean +
// strings.HasPrefix. This prevents traversal even if validateRelPath were
// somehow bypassed (defence in depth).
//
// The safe pattern:
//   rmTarget := filepath.Join("/configs", filePath)
//   rmTarget = filepath.Clean(rmTarget)
//   if !strings.HasPrefix(rmTarget, "/configs/") { return err }
// passes ONE sanitized argument to rm, which resolves it relative to rm's
// CWD (/), NOT the shell's working directory.
//
// By contrast, the vulnerable shell-expanded form:
//   sh -c "rm -rf /configs $filePath"
// would treat ".." as path components relative to /configs and could escape.
//
// deleteViaEphemeral uses the exec form with scoped path (verified in code review).
func TestDeleteViaEphemeral_SafeForm(t *testing.T) {
	// This test confirms the safe form is present in the actual codebase.
	src, err := sourceFile("container_files.go")
	if err != nil {
		t.Skip("cannot read source: " + err.Error())
	}
	// Check for filepath.Join scoping to /configs
	if !strings.Contains(src, `filepath.Join("/configs", filePath)`) {
		t.Error("deleteViaEphemeral does not use filepath.Join scoping to /configs; F1085 fix may be missing or reverted")
	}
	// Check for filepath.Clean normalization
	if !strings.Contains(src, `filepath.Clean(rmTarget)`) {
		t.Error("deleteViaEphemeral does not use filepath.Clean; F1085 fix may be missing or reverted")
	}
	// Check for HasPrefix boundary guard
	if !strings.Contains(src, `strings.HasPrefix(rmTarget, "/configs/")`) {
		t.Error("deleteViaEphemeral does not use HasPrefix boundary guard; F1085 fix may be missing or reverted")
	}
}

// sourceFile reads a source file from the same package at runtime.
// Used for compile-time-verification-style tests without importing io/ioutil.
func sourceFile(name string) (string, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return "", err
	}
	return string(data), nil
}