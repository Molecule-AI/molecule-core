package handlers

import (
	"testing"
)

// ---- CWE-22 regression tests ----
//
// Vulnerability: the tar header used a raw, unvalidated name. A name like
//   "foo/../../../etc"
// passes a "no leading .." prefix check but resolves to "/etc" after
// filepath.Join with any base, escaping the intended directory.
//
// Fix (PR #1434): re-validate archiveName using filepath.Clean before
// writing the tar header, blocking absolute paths and any path that
// resolves to "..".  The same guard logic is exposed via validateRelPathJoined
// so callers can re-validate after their own Join.

func TestValidateRelPath_CWE22(t *testing.T) {
	cases := []struct {
		name    string
		relPath string
		wantErr bool
	}{
		// ---- allowed ----
		{"clean relative path", "config/app.yaml", false},
		{"clean subdirectory", "data/logs/server.log", false},
		{"dot file", ".", false},
		{"dot slash file", "./file", false},
		{"sibling file", "../sibling/file", false}, // still relative, does not escape
		{"normal filename", "my-agent/SKILL.md", false},
		// ---- blocked ----
		{"leading ..", "../secret", true},
		{"leading .. /", "../", true},
		{"just ..", "..", true},
		{"embedded traversal", "foo/../../../etc", true},
		{"embedded traversal 2", "../../../etc/passwd", true},
		{"absolute path", "/etc/passwd", true},
		{"absolute shadow", "/etc/shadow", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRelPath(tc.relPath)
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateRelPathJoined_CWE22(t *testing.T) {
	// validateRelPathJoined adds a post-Join revalidation: the caller does
	//   joined := filepath.Join(destDir, clean)
	//   // then calls validateRelPathJoined to check the result
	// The fix catches traversal that "foo/../../../etc" can still achieve
	// even after a pre-check, because Join normalises the path.
	cases := []struct {
		name    string
		relPath string
		destDir string
		wantErr bool
	}{
		// ---- allowed ----
		{"clean relative", "foo/bar.yaml", "/configs", false},
		{"subdirectory", "dir/sub/file.txt", "/data", false},
		{"dot", ".", "/base", false},
		{"dot slash", "./file", "/base", false},
		// ---- blocked ----
		{"single ..", "..", "/base", true},
		{"leading ..", "../foo", "/base", true},
		{"embedded escapes", "foo/../../../etc", "/configs", true},
		{"deep escape", "../../../etc/passwd", "/data", true},
		{"absolute path", "/etc/passwd", "/base", true},
		// key CWE-22 case: passes pre-join check but post-join /configs/../../../etc = /etc
		{"post-join escape", "foo/../../../etc", "/configs", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRelPathJoined(tc.relPath, tc.destDir)
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
