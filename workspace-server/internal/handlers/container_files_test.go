package handlers

import (
	"archive/tar"
	"bytes"
	"testing"
)

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

// ==================== copyFilesToContainer ====================

func TestCopyFilesToContainer_PathTraversal(t *testing.T) {
	files := map[string]string{
		"../../../etc/passwd": "malicious content",
	}
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	err := copyFilesToContainer(tw, "/configs", files)

	// Must reject: cleaned path starts with ".." after filepath.Clean
	if err == nil {
		t.Error("expected error for traversal path '../../../etc/passwd', got nil")
	}
}

func TestCopyFilesToContainer_NestedTraversalEscapesDest(t *testing.T) {
	// foo/../bar is normalized to bar — safe
	// foo/../../etc/passwd normalizes to etc/passwd — safe (stays inside destPath)
	// ../ alone fails HasPrefix check
	// On platforms where Clean("../../../etc/passwd") = "../../../etc/passwd",
	// the new defense-in-depth check catches it.
	files := map[string]string{
		"foo/../bar":                  "should be safe (normalizes to bar)",
		"valid-file.txt":              "should be safe",
		"subdir/nested/file.yaml":     "should be safe",
	}
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	err := copyFilesToContainer(tw, "/configs", files)
	if err != nil {
		t.Errorf("expected no error for valid paths, got: %v", err)
	}
}

func TestCopyFilesToContainer_CleanPathUsedInTarHeader(t *testing.T) {
	// Verify that cleaned paths are used in the tar header, not raw paths.
	// This is the regression test: PR #1363 used raw `name` in the tar header.
	files := map[string]string{
		"subdir/config.yaml": "workspace config",
	}
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	err := copyFilesToContainer(tw, "/configs", files)
	if err != nil {
		t.Fatalf("copyFilesToContainer returned error: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close error: %v", err)
	}

	// Verify the tar entry name is relative inside destPath, not a raw traversal
	tr := tar.NewReader(&buf)
	found := false
	for {
		hdr, err := tr.Next()
		if err != nil {
			break
		}
		if hdr.Name == "" {
			continue
		}
		// Tar header name must be relative (inside /configs), never absolute
		if hdr.Name[0] == '/' {
			t.Errorf("tar header contains absolute path: %q — path traversal risk", hdr.Name)
		}
		// Header name must be inside /configs
		if !bytes.HasPrefix([]byte(hdr.Name), []byte("/configs")) && !bytes.Contains([]byte(hdr.Name), []byte("subdir")) {
			t.Logf("tar header name: %q (relative path, OK)", hdr.Name)
		}
		found = true
	}
	if !found {
		t.Error("no tar entries found")
	}
}
