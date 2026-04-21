package handlers

import (
	"testing"
)

// container_files.go defines methods on TemplatesHandler that interact with Docker.
// These tests cover the path-validation and tar-building logic that can be tested
// without a Docker daemon.

// ---------- copyFilesToContainer: rejects absolute paths ----------

func TestCopyFilesToContainer_RejectsAbsolutePath(t *testing.T) {
	h := &TemplatesHandler{} // docker is nil — won't reach Docker calls

	err := h.copyFilesToContainer(t.Context(), "dummy-container", "/configs", map[string]string{
		"/etc/passwd": "hacked",
	})
	if err == nil {
		t.Fatal("expected error for absolute path, got nil")
	}
	if got := err.Error(); got != "unsafe file path in archive: /etc/passwd" {
		t.Errorf("unexpected error message: %s", got)
	}
}

// ---------- copyFilesToContainer: rejects path traversal ----------

func TestCopyFilesToContainer_RejectsTraversal(t *testing.T) {
	h := &TemplatesHandler{}

	err := h.copyFilesToContainer(t.Context(), "dummy-container", "/configs", map[string]string{
		"../../etc/shadow": "hacked",
	})
	if err == nil {
		t.Fatal("expected error for traversal path, got nil")
	}
}

// ---------- copyFilesToContainer: accepts valid relative paths ----------

func TestCopyFilesToContainer_AcceptsRelativePath(t *testing.T) {
	h := &TemplatesHandler{}

	// Without a Docker client, CopyToContainer will fail — but we verify
	// that the tar-building phase succeeds (no "unsafe file path" error).
	err := h.copyFilesToContainer(t.Context(), "dummy-container", "/configs", map[string]string{
		"my-plugin/config.yaml": "name: test",
		"another-file.txt":      "hello",
	})
	// Should fail at Docker call, not at path validation
	if err == nil {
		t.Fatal("expected Docker error (nil client), got nil")
	}
	// The error should be a Docker/nil-pointer error, not a path validation error
	if got := err.Error(); got == "unsafe file path in archive: my-plugin/config.yaml" {
		t.Error("valid path was incorrectly rejected as unsafe")
	}
}

// ---------- findContainer: nil docker → empty string ----------

func TestFindContainer_NilDocker(t *testing.T) {
	h := &TemplatesHandler{} // docker is nil

	result := h.findContainer(t.Context(), "ws-123")
	if result != "" {
		t.Errorf("expected empty string for nil docker, got %q", result)
	}
}

// ---------- writeViaEphemeral: nil docker → error ----------

func TestWriteViaEphemeral_NilDocker(t *testing.T) {
	h := &TemplatesHandler{}

	err := h.writeViaEphemeral(t.Context(), "vol-123", map[string]string{
		"test.txt": "content",
	})
	if err == nil {
		t.Fatal("expected error for nil docker, got nil")
	}
	if got := err.Error(); got != "docker not available" {
		t.Errorf("expected 'docker not available', got %q", got)
	}
}

// ---------- deleteViaEphemeral: nil docker → error ----------

func TestDeleteViaEphemeral_NilDocker(t *testing.T) {
	h := &TemplatesHandler{}

	err := h.deleteViaEphemeral(t.Context(), "vol-123", "test.txt")
	if err == nil {
		t.Fatal("expected error for nil docker, got nil")
	}
	if got := err.Error(); got != "docker not available" {
		t.Errorf("expected 'docker not available', got %q", got)
	}
}
