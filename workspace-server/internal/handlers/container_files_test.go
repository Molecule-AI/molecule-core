package handlers

import (
	"archive/tar"
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// copyFilesToContainerHarness mirrors the sanitised tar-generation logic from
// copyFilesToContainer so we can test path validation in isolation (no Docker).
func copyFilesToContainerHarness(destPath string, files map[string]string) (string, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	createdDirs := map[string]bool{}
	for name, content := range files {
		// CWE-22: validate before use.
		if err := validateRelPath(name); err != nil {
			return "", err
		}
		cleanName := filepath.Clean(name)
		archiveName := filepath.Join(destPath, cleanName)
		// Defence-in-depth: joined path must not escape destPath.
		if !strings.HasPrefix(archiveName, destPath) && archiveName != destPath {
			return "", fmt.Errorf("path escapes destination: %s", name)
		}

		dir := filepath.Dir(archiveName)
		if dir != "." && dir != destPath && !createdDirs[dir] {
			tw.WriteHeader(&tar.Header{
				Typeflag: tar.TypeDir,
				Name:     dir + "/",
				Mode:     0755,
			})
			createdDirs[dir] = true
		}

		data := []byte(content)
		header := &tar.Header{
			Name: archiveName,
			Mode: 0644,
			Size: int64(len(data)),
		}
		if err := tw.WriteHeader(header); err != nil {
			return "", err
		}
		if _, err := tw.Write(data); err != nil {
			return "", err
		}
	}
	if err := tw.Close(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ==================== Valid paths (must pass) ====================

func TestCopyFilesToContainer_ValidPaths(t *testing.T) {
	t.Parallel()
	cases := []string{
		"foo.txt",
		"foo/bar.txt",
		"foo/bar/baz.txt",
		"a",
		"foo-bar_baz",
		"123",
		".hidden",
		"foo/bar/baz/qux.txt",
	}
	for _, p := range cases {
		p := p
		t.Run(p, func(t *testing.T) {
			t.Parallel()
			_, err := copyFilesToContainerHarness("/configs", map[string]string{p: "data"})
			if err != nil {
				t.Errorf("valid path %q returned unexpected error: %v", p, err)
			}
		})
	}
}

func TestCopyFilesToContainer_NestedValidPath(t *testing.T) {
	t.Parallel()
	_, err := copyFilesToContainerHarness("/configs", map[string]string{
		"foo/bar/baz/qux.txt": "nested content",
	})
	if err != nil {
		t.Errorf("expected no error for nested valid path, got: %v", err)
	}
}

// ==================== Absolute paths — Linux-style must reject ====================

func TestCopyFilesToContainer_LinuxAbsolutePath(t *testing.T) {
	t.Parallel()
	// Unix absolute paths must be rejected.
	unsafe := []string{
		"/etc/passwd",
		"/configs/foo",
		"/",
	}
	for _, p := range unsafe {
		p := p
		t.Run(p, func(t *testing.T) {
			t.Parallel()
			_, err := copyFilesToContainerHarness("/configs", map[string]string{p: "malicious"})
			if err == nil {
				t.Errorf("expected error for absolute path %q, got nil", p)
			}
		})
	}
}

// ==================== Leading ".." prefix (must reject) ====================

func TestCopyFilesToContainer_LeadingDotDot(t *testing.T) {
	t.Parallel()
	// Paths that start with ".." in their cleaned form are rejected.
	unsafe := []string{
		"../etc/passwd",
		"foo/../../etc/passwd",
		"..",
		"../",
	}
	for _, p := range unsafe {
		p := p
		t.Run(p, func(t *testing.T) {
			t.Parallel()
			_, err := copyFilesToContainerHarness("/configs", map[string]string{p: "malicious"})
			if err == nil {
				t.Errorf("expected error for leading '..' path %q, got nil", p)
			}
		})
	}
}

// ==================== Mid-path traversal — the regression case (must reject) ====================

func TestCopyFilesToContainer_MidPathTraversal(t *testing.T) {
	t.Parallel()
	// "foo/../../../etc" passes the leading ".." check (cleaned to "../../etc")
	// but after filepath.Join("/configs", "../../etc") resolves to "/etc",
	// escaping the volume mount. This is the primary CWE-22 regression vector.
	_, err := copyFilesToContainerHarness("/configs", map[string]string{
		"foo/../../../etc/passwd": "malicious",
	})
	if err == nil {
		t.Errorf("expected error for mid-path traversal 'foo/../../../etc/passwd', got nil")
	}
}

func TestCopyFilesToContainer_MidPathTraversalVarious(t *testing.T) {
	t.Parallel()
	// These paths resolve to locations OUTSIDE /configs after filepath.Join,
	// so validateRelPath must block them before they reach the tar header.
	cases := []string{
		"a/b/../../../etc",
		"x/y/z/../../../../etc/passwd",
		"foo/bar/../../../../../../../etc/shadow",
	}
	for _, p := range cases {
		p := p
		t.Run(p, func(t *testing.T) {
			t.Parallel()
			_, err := copyFilesToContainerHarness("/configs", map[string]string{p: "malicious"})
			if err == nil {
				t.Errorf("expected error for mid-path traversal %q, got nil", p)
			}
		})
	}
}

// ==================== DotDot cleaned paths — safe, must pass ====================

func TestCopyFilesToContainer_DotDotCleanedPath(t *testing.T) {
	t.Parallel()
	// filepath.Clean normalises these before the ".." check. "foo..bar" and "a..b"
	// are safe because the dots are adjacent. "..." becomes "../.." and is
	// correctly rejected by validateRelPath (cleaned to "../..", starts with "..").
	safe := []string{
		"foo..bar",
		"a..b",
		"foo/../bar",   // cleaned to "bar" — no leading "..", stays inside dest
		"foo/..",       // cleaned to "."  — no leading "..", stays inside dest
		"aaa/bbb/ccc/../../../ddd", // cleaned to "ddd" — no leading "..", safe
	}
	for _, p := range safe {
		_, err := copyFilesToContainerHarness("/configs", map[string]string{p: "data"})
		if err != nil {
			t.Errorf("expected no error for dotdot-cleaned path %q, got: %v", p, err)
		}
	}
	// "..." normalises to "../.." which starts with "..", so validateRelPath
	// correctly rejects it — confirm this is the expected behaviour.
	_, err := copyFilesToContainerHarness("/configs", map[string]string{"...": "data"})
	if err == nil {
		t.Error("expected '...' to be rejected (cleaned to '../..'), got nil")
	}
}

// ==================== Defence-in-depth: legitimate paths produce correct tar headers ====================

func TestCopyFilesToContainer_JoinedPathStaysInsideDest(t *testing.T) {
	t.Parallel()
	// Confirm legitimate paths produce tar entries with paths inside /configs.
	archive, err := copyFilesToContainerHarness("/configs", map[string]string{
		"foo/bar.txt": "legitimate",
	})
	if err != nil {
		t.Fatalf("legitimate path should not error: %v", err)
	}
	tr := tar.NewReader(bytes.NewReader([]byte(archive)))
	found := false
	for {
		hdr, err := tr.Next()
		if err != nil {
			break
		}
		if strings.HasPrefix(hdr.Name, "/configs/foo") || strings.HasPrefix(hdr.Name, "configs/foo") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected tar entry under /configs, got archive bytes (len=%d)", len(archive))
	}
}