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
