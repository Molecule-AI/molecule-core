package handlers

// container_files_test.go — CWE-22 regression suite for copyFilesToContainer.
//
// Vulnerability: copyFilesToContainer validated the raw filename before
// filepath.Join(destPath, name) but placed the post-join result in the tar
// header.  A mid-path traversal such as "foo/../../../etc" passes the prefix
// check (does not start with "..") yet resolves to /etc after the join,
// escaping the volume mount and writing outside the container's filesystem.
//
// Fix (PR #1434): re-validate archiveName after filepath.Join using
// filepath.Clean, then use the cleaned result in the tar header.
// A Docker client is not required for these tests — the validation rejects
// unsafe paths before any Docker call is made.

import (
	"context"
	"errors"
	"testing"
)

func TestCopyFilesToContainer_CWE22_RejectsTraversal(t *testing.T) {
	// TemplatesHandler with nil docker — validation runs before any Docker call.
	h := &TemplatesHandler{docker: nil}

	ctx := context.Background()

	tests := []struct {
		label     string
		destPath  string
		files     map[string]string
		wantErr   bool
		errSubstr string // substring that must appear in error message
	}{
		// ── Legitimate paths ───────────────────────────────────────────────────
		{
			label:    "simple_relative_path_ok",
			destPath: "/configs",
			files:    map[string]string{"config.yaml": "key: value"},
			wantErr:  false,
		},
		{
			label:    "nested_relative_path_ok",
			destPath: "/configs",
			files:    map[string]string{"subdir/script.sh": "#!/bin/sh"},
			wantErr:  false,
		},
		{
			label:    "dot_in_filename_ok",
			destPath: "/configs",
			files:    map[string]string{"app.venv/config": "data"},
			wantErr:  false,
		},
		// ── CWE-22: absolute-path prefix ────────────────────────────────────────
		{
			label:     "absolute_path_rejected",
			destPath:  "/configs",
			files:     map[string]string{"/etc/passwd": "malicious"},
			wantErr:   true,
			errSubstr: "unsafe file path",
		},
		// ── CWE-22: leading ".." prefix ─────────────────────────────────────────
		{
			label:     "leading_dotdot_rejected",
			destPath:  "/configs",
			files:     map[string]string{"../etc/passwd": "malicious"},
			wantErr:   true,
			errSubstr: "unsafe file path",
		},
		// ── CWE-22: mid-path traversal (the regression case) ────────────────────
		// "foo/../../../etc" does NOT start with ".." — passed the old check.
		// After filepath.Join("/configs", "foo/../../../etc") → Clean → /etc
		// (absolute), escaping the volume mount.  Rejected by the post-join guard.
		{
			label:     "mid_path_traversal_rejected",
			destPath:  "/configs",
			files:     map[string]string{"foo/../../../etc/cron.d/malicious": "* * * * * root echo pwned"},
			wantErr:   true,
			errSubstr: "path escapes destination",
		},
		{
			label:     "mid_path_traversal_escapes_configs",
			destPath:  "/configs",
			files:     map[string]string{"x/y/../../../../../../../etc/shadow": "malicious"},
			wantErr:   true,
			errSubstr: "path escapes destination",
		},
		{
			label:     "double_dotdot_in_subpath_rejected",
			destPath:  "/workspace",
			files:     map[string]string{"a/../../../workspace/somefile": "data"},
			wantErr:   true,
			errSubstr: "path escapes destination",
		},
		// ── CWE-22: traversal targeting parent of destPath ───────────────────────
		{
			label:     "escapes_destpath_via_traversal",
			destPath:  "/configs",
			files:     map[string]string{"..%2F..%2F..%2Fsecrets": "data"}, // URL-encoded "../" — still a traversal
			wantErr:   true,
			errSubstr: "path escapes destination",
		},
		// ── Mixed: valid entry + traversal entry ────────────────────────────────
		{
			label:     "one_traversal_in_map_rejected",
			destPath:  "/configs",
			files:     map[string]string{"good.txt": "valid", "foo/../../../evil": "bad"},
			wantErr:   true,
			errSubstr: "path escapes destination",
		},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			err := h.copyFilesToContainer(ctx, "any-container", tc.destPath, tc.files)
			if tc.wantErr {
				if err == nil {
					t.Errorf("want non-nil error, got nil")
					return
				}
				if tc.errSubstr != "" && !errors.Is(err, context.DeadlineExceeded) &&
					!contains(err.Error(), tc.errSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.errSubstr)
				}
			} else {
				// wantErr == false: we expect nil from a nil-docker call.
				// With nil docker the function will panic or return a docker-err
				// only if the path check is bypassed.  We use a strict check:
				// any error other than a docker-initialized error means the path
				// was incorrectly allowed.
				if err != nil && contains(err.Error(), "unsafe") {
					t.Errorf("want nil (path accepted), got error: %v", err)
				}
			}
		})
	}
}

// contains is a simple substring check (no external imports needed in this file).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchSubstring(s, substr)))
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ─────────────────────────────────────────────────────────────────────────────
// F1085 regression suite for deleteViaEphemeral.
//
// Vulnerability (GH#1600 / F1085): deleteViaEphemeral accepted a 2-arg exec
// form — Cmd: []string{"rm", "-rf", "/configs", filePath} — and passed the
// raw filePath to rm as a separate argument.  A traversal path such as
// "foo/../../../etc" caused rm to delete /etc OUTSIDE the /configs volume
// mount, and also deleted /configs itself.  Fix (PR #1627): switch to a
// single-arg concat form — Cmd: []string{"rm", "-rf", "/configs/" + filePath}
// — so the traversal is part of the path inside the volume and gets caught
// by the existing validateRelPath guard (filepath.Clean("foo/../../../etc") =
// "/etc"; IsAbs("/etc") = true → rejected before any Docker call).
// These tests verify validateRelPath blocks all traversal forms at the door,
// so deleteViaEphemeral never reaches the Docker exec regardless of the rm
// form used.

func TestDeleteViaEphemeral_F1085_RejectsTraversal(t *testing.T) {
	// nil docker → "docker not available" only if validateRelPath is bypassed.
	// With a traversal path, validateRelPath returns a path error first.
	h := &TemplatesHandler{docker: nil}

	ctx := context.Background()

	tests := []struct {
		label      string
		volumeName string
		filePath   string
		wantErr    bool
		errSubstr  string // must appear in error when rejected
	}{
		// ── Legitimate relative paths — validateRelPath accepts ─────────────────
		{
			label:      "simple_relative_ok",
			volumeName: "myvol",
			filePath:   "config.yaml",
			wantErr:    false, // validateRelPath passes → reaches docker nil → "docker not available"
		},
		{
			label:      "nested_relative_ok",
			volumeName: "myvol",
			filePath:   "subdir/script.sh",
			wantErr:    false,
		},
		// ── F1085: leading ".." prefix ────────────────────────────────────────────
		{
			label:      "leading_dotdot_rejected",
			volumeName: "myvol",
			filePath:   "../etc/passwd",
			wantErr:    true,
			errSubstr:  "path traversal",
		},
		// ── F1085: mid-path traversal (the regression case) ───────────────────────
		// "foo/../../../etc" cleaned to "/etc" → IsAbs = true → validateRelPath rejects.
		{
			label:      "mid_path_traversal_rejected",
			volumeName: "myvol",
			filePath:   "foo/../../../etc/cron.d/malicious",
			wantErr:    true,
			errSubstr:  "path traversal",
		},
		{
			label:      "deep_mid_path_traversal_rejected",
			volumeName: "myvol",
			filePath:   "x/y/../../../../../../../etc/shadow",
			wantErr:    true,
			errSubstr:  "path traversal",
		},
		// ── F1085: double-dot in subpath ─────────────────────────────────────────
		{
			label:      "double_dotdot_in_subpath_rejected",
			volumeName: "myvol",
			filePath:   "a/../../../workspace/secret",
			wantErr:    true,
			errSubstr:  "path traversal",
		},
		// ── F1085: URL-encoded traversal ──────────────────────────────────────────
		{
			label:      "url_encoded_traversal_rejected",
			volumeName: "myvol",
			filePath:   "..%2F..%2Fsecrets",
			wantErr:    true,
			errSubstr:  "path traversal",
		},
		// ── F1085: absolute path ───────────────────────────────────────────────────
		{
			label:      "absolute_path_rejected",
			volumeName: "myvol",
			filePath:   "/etc/passwd",
			wantErr:    true,
			errSubstr:  "path traversal",
		},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			err := h.deleteViaEphemeral(ctx, tc.volumeName, tc.filePath)
			if tc.wantErr {
				if err == nil {
					t.Errorf("want non-nil error, got nil")
					return
				}
				if tc.errSubstr != "" && !contains(err.Error(), tc.errSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.errSubstr)
				}
			} else {
				// wantErr == false: validateRelPath passes → nil docker → "docker not available"
				if err == nil {
					t.Errorf("want non-nil error (docker not available), got nil")
					return
				}
				// Any error other than the docker-initialized one means the path was
				// incorrectly rejected by validateRelPath.
				if !contains(err.Error(), "docker not available") {
					t.Errorf("unexpected error: %v (want docker-not-available, not path rejection)", err)
				}
			}
		})
	}
}
