package plugins

// supply_chain.go — plugin content-integrity verification (issue #768, VULN-004).
//
// VerifyManifestIntegrity is the only exported symbol. It is called from
// resolveAndStage() in platform/internal/handlers/plugins_install_pipeline.go
// after every fetch, before the staged plugin reaches deliverToContainer.
//
// Algorithm (deterministic across OSes):
//  1. Walk all regular files in stagedDir, skipping "manifest.json" itself.
//  2. For each file build the string "<rel-path>\x00<file-content>".
//  3. Sort the strings lexicographically by rel-path.
//  4. SHA256-hash the concatenated sorted strings.
//  5. Compare (case-insensitive) to the "sha256" field in manifest.json.
//
// This is the same algorithm used by supply_chain_test.go's stagedDirDigest
// helper — they must stay in sync so tests remain deterministic.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// VerifyManifestIntegrity checks whether the staged plugin tree matches the
// SHA256 digest declared in manifest.json (if present).
//
// Behaviour contract:
//   - manifest.json absent                     → nil (backward-compatible)
//   - manifest.json present, no sha256 field   → nil (backward-compatible)
//   - sha256 field matches computed digest      → nil
//   - sha256 field does NOT match              → non-nil error
//
// DB errors in the caller are fail-open; VerifyManifestIntegrity itself is
// fail-closed — a read error on any staged file returns a non-nil error so
// a partially-written tree is never silently accepted.
func VerifyManifestIntegrity(stagedDir string) error {
	manifestPath := filepath.Join(stagedDir, "manifest.json")

	// No manifest.json — skip check (backward-compatible with plugins that
	// pre-date supply-chain hardening).
	data, err := os.ReadFile(manifestPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("manifest integrity: read manifest.json: %w", err)
	}

	// Parse the manifest JSON. Unknown keys are ignored.
	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("manifest integrity: parse manifest.json: %w", err)
	}

	// If the manifest carries no sha256 field (or it is not a string), skip
	// the check. This preserves backward compatibility with manifests that
	// store only metadata (name, version, …) but no integrity digest.
	expectedHex, ok := manifest["sha256"].(string)
	if !ok || expectedHex == "" {
		return nil
	}

	// Compute the canonical digest of the staged tree (excluding manifest.json).
	gotHex, err := computeStagedDirDigest(stagedDir)
	if err != nil {
		return fmt.Errorf("manifest integrity: compute digest: %w", err)
	}

	if !strings.EqualFold(gotHex, expectedHex) {
		return fmt.Errorf(
			"manifest integrity: sha256 mismatch — manifest declares %s but computed %s; "+
				"plugin content may be tampered or corrupted",
			expectedHex, gotHex,
		)
	}
	return nil
}

// computeStagedDirDigest walks dir, hashes all regular files (excluding
// "manifest.json"), and returns the lower-case hex SHA256 digest.
//
// The algorithm is intentionally stable across operating systems: we sort
// file entries lexicographically so the result is independent of filesystem
// iteration order (which differs between ext4 and HFS+, for example).
func computeStagedDirDigest(dir string) (string, error) {
	var entries []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		// Exclude the manifest from its own verification — it is the
		// verifier, not part of the verified content.
		if rel == "manifest.json" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		// Build "<rel-path>\x00<content>" as a single string so that a file
		// whose content matches another file's path doesn't produce a
		// collision.
		entries = append(entries, rel+"\x00"+string(content))
		return nil
	})
	if err != nil {
		return "", err
	}

	sort.Strings(entries)
	sum := sha256.Sum256([]byte(strings.Join(entries, "")))
	return hex.EncodeToString(sum[:]), nil
}
