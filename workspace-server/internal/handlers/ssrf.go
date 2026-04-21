package handlers

import (
	"fmt"
	"path/filepath"
	"strings"
)

// validateRelPath checks that a file path is relative and does not escape
// the destination via absolute paths or ".." traversal. Used by
// copyFilesToContainer and deleteViaEphemeral as a defence-in-depth measure.
func validateRelPath(filePath string) error {
	clean := filepath.Clean(filePath)
	if filepath.IsAbs(clean) || strings.Contains(clean, "..") {
		return fmt.Errorf("path traversal or absolute path not allowed: %s", filePath)
	}
	return nil
}