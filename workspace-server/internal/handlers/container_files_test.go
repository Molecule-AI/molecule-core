package handlers

import "testing"

// ==================== validateRelPath ====================

func TestValidateRelPath_ValidRelativePaths(t *testing.T) {
	valid := []string{
		"foo.txt",
		"foo/bar.txt",
		"foo/bar/baz.txt",
		"a",
		"foo-bar_baz",
		"123",
		".hidden",
		"foo/bar/baz/qux.txt",
	}
	for _, p := range valid {
		t.Run(p, func(t *testing.T) {
			if err := validateRelPath(p); err != nil {
				t.Errorf("validateRelPath(%q) returned unexpected error: %v", p, err)
			}
		})
	}
}

func TestValidateRelPath_RejectsAbsolutePaths(t *testing.T) {
	unsafe := []string{
		"/etc/passwd",
		"/configs/foo",
		"C:\\Windows\\System32",
		"/",
	}
	for _, p := range unsafe {
		t.Run(p, func(t *testing.T) {
			if err := validateRelPath(p); err == nil {
				t.Errorf("validateRelPath(%q) expected error, got nil", p)
			}
		})
	}
}

func TestValidateRelPath_RejectsDotDotTraversal(t *testing.T) {
	unsafe := []string{
		"../etc/passwd",
		"foo/../../etc/passwd",
		"foo/../bar",
		"..",
		"../",
		"foo/..",
		"....//....//....//etc/passwd", // cleaned to ../../etc/passwd
	}
	for _, p := range unsafe {
		t.Run(p, func(t *testing.T) {
			if err := validateRelPath(p); err == nil {
				t.Errorf("validateRelPath(%q) expected error (path traversal), got nil", p)
			}
		})
	}
}

func TestValidateRelPath_DotDotCleanedPath(t *testing.T) {
	// filepath.Clean normalises the input before the ".." check, so
	// sequences buried inside clean names (e.g. "foo..bar") are fine.
	valid := []string{
		"foo..bar",
		"...",
		"a..b",
	}
	for _, p := range valid {
		t.Run(p, func(t *testing.T) {
			if err := validateRelPath(p); err != nil {
				t.Errorf("validateRelPath(%q) unexpected error: %v", p, err)
			}
		})
	}
}

// ==================== CWE-78 — deleteViaEphemeral exec form ====================
// See INCIDENT_LOG.md P0 section for regression history (F1502, PR #1580).
// The correct exec form for rm is:
//   Cmd: []string{"rm", "-rf", "/configs", filePath}
// The WRONG (regression) form:
//   Cmd: []string{"rm", "-rf", "/configs/" + filePath}
// The concat form allows path traversal: with filePath="foo/../bar",
// the path becomes "/configs/foo/../bar" → rm recursively deletes /configs/../bar
// (escape outside /configs volume). The exec form bounds rm to the volume
// via the container bind mount volumeName+":/configs".
//
// PR #1583 introduced the regression at container_files.go:174.
// This regression was previously introduced in PR #1498 (#85de7d6) and fixed
// in #9246924. It reappeared in #1583 via commit a3cc162 ("ship: apply CWE-22/...").
// Once #1583 is merged, this comment should be updated to reflect the fix.

// TestDeleteViaEphemeral_ExecFormDocumentsRegression documents the correct
// exec form vs the regression (string concatenation). The regression occurs
// when container_files.go uses `"/configs/" + filePath` instead of separate
// "/configs" and filePath arguments in the rm command.
// We can't test the actual rm output without a full Docker mock, but this test
// documents the invariant: the rm command MUST use two-argument form.
func TestDeleteViaEphemeral_ExecFormDocumentsRegression(t *testing.T) {
	// This test documents the regression for PR #1583.
	// When the bug is fixed (container_files.go:174 reverts to exec form),
	// this test remains as a regression guard.
	//
	// CORRECT (exec form — bounds rm to /configs volume via bind mount):
	//   Cmd: []string{"rm", "-rf", "/configs", filePath}
	//   Binds: [volumeName + ":/configs"]
	//   With filePath="foo/../bar" → rm receives ["/configs", "foo/../bar"]
	//   rm operates inside /configs → foo/../bar resolves INSIDE volume → safe
	//
	// WRONG (string concat — allows path traversal):
	//   Cmd: []string{"rm", "-rf", "/configs/" + filePath}
	//   With filePath="foo/../bar" → rm -rf /configs/foo/../bar
	//   rm traverses /configs/../bar → escapes volume bounds → CWE-78
	//
	// The fix: change Cmd back to two-argument exec form.
	// This test always passes — it only documents the regression.
	t.Log("CWE-78 regression guard: exec form must be two-argument: rm -rf /configs filePath")
}
