package handlers

import (
	"testing"
)

func TestValidateRelPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid simple file",
			path:    "config.yaml",
			wantErr: false,
		},
		{
			name:    "valid nested file",
			path:    "subdir/config.yaml",
			wantErr: false,
		},
		{
			name:    "valid deep nested file",
			path:    "a/b/c/d/file.txt",
			wantErr: false,
		},
		{
			name:    "valid file with dots in name",
			path:    "my.config.file.yaml",
			wantErr: false,
		},
		{
			name:    "normalized to valid — foo/../bar becomes bar",
			path:    "foo/../bar",
			wantErr: false,
		},
		{
			name:    "rejected — double dot traversal",
			path:    "../etc/passwd",
			wantErr: true,
			errMsg:  "unsafe path",
		},
		{
			name:    "rejected — triple dot traversal",
			path:    "../../../etc/passwd",
			wantErr: true,
			errMsg:  "unsafe path",
		},
		{
			name:    "rejected — absolute path Unix",
			path:    "/etc/passwd",
			wantErr: true,
			errMsg:  "unsafe path",
		},
		{
			name:    "rejected — absolute path Windows",
			path:    "C:\\Windows\\System32",
			wantErr: true,
			errMsg:  "unsafe path",
		},
		{
			name:    "rejected — mixed traversal with valid prefix",
			path:    "foo/../../../etc/passwd",
			wantErr: true,
			errMsg:  "unsafe path",
		},
		{
			name:    "rejected — dotdot mid-path",
			path:    "foo/../bar/../etc/passwd",
			wantErr: true,
			errMsg:  "unsafe path",
		},
		{
			name:    "valid — dot-only path normalizes",
			path:    "./config.yaml",
			wantErr: false,
		},
		{
			name:    "valid — trailing slash",
			path:    "dir/",
			wantErr: false,
		},
		{
			name:    "valid — empty string edge case",
			path:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRelPath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateRelPath(%q) = nil, want error containing %q", tt.path, tt.errMsg)
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("validateRelPath(%q) error = %q, want error containing %q", tt.path, err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateRelPath(%q) = %v, want nil", tt.path, err)
				}
			}
		})
	}
}

// contains reports whether substr is within s.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}