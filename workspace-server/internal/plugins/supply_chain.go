package plugins

// supply_chain.go — plugin supply-chain integrity controls (issue #768 / #815).
//
// Two controls:
//
//  1. VerifyManifestIntegrity — SHA256 content-integrity check.  If a
//     staged plugin directory contains a manifest.json with a "sha256"
//     field, the field is compared against the canonical digest of the
//     staged tree.  A mismatch aborts the install before any files reach
//     a workspace volume.
//
//  2. Pinned-ref gate — enforced in GithubResolver.Fetch (github.go).
//     A bare "org/repo" spec with no "#tag" or "#sha" fragment is rejected
//     unless PLUGIN_ALLOW_UNPINNED=true (local-dev escape hatch).

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// VerifyManifestIntegrity checks the SHA256 content hash declared in
// manifest.json against the actual contents of stagedDir.
//
// Behaviour:
//   - manifest.json absent           → nil (backward compat with pre-#768 plugins)
//   - manifest.json present, no sha256 field → nil (same backward compat)
//   - sha256 field matches computed digest  → nil (integrity verified)
//   - sha256 field doesn't match           → non-nil error (tamper detected)
//
// The canonical digest algorithm walks all regular files in stagedDir
// (excluding manifest.json itself), sorts by relative path, and SHA256-hashes
// the concatenation of "<rel-path>\x00<file-content>" strings. This matches
// the reference implementation in supply_chain_test.go:stagedDirDigest.
func VerifyManifestIntegrity(stagedDir string) error {
	manifestPath := filepath.Join(stagedDir, "manifest.json")

	data, err := os.ReadFile(manifestPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil // no manifest — backward compat, skip check
	}
	if err != nil {
		return fmt.Errorf("supply chain: read manifest.json: %w", err)
	}

	// Parse only the sha256 field; ignore other metadata keys.
	var manifest struct {
		SHA256 string `json:"sha256"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("supply chain: parse manifest.json: %w", err)
	}
	if manifest.SHA256 == "" {
		return nil // no sha256 field — backward compat, skip check
	}

	computed, err := stagedTreeDigest(stagedDir)
	if err != nil {
		return fmt.Errorf("supply chain: compute tree digest: %w", err)
	}

	if computed != manifest.SHA256 {
		return fmt.Errorf("supply chain: sha256 mismatch — manifest claims %s, computed %s; plugin may have been tampered with",
			manifest.SHA256, computed)
	}
	return nil
}

// stagedTreeDigest computes the canonical SHA256 of all regular files in
// stagedDir, excluding manifest.json. Algorithm:
//
//  1. Walk all regular files, skipping manifest.json.
//  2. For each file, build the string "<rel-path>\x00<file-content>".
//  3. Sort lexicographically by relative path.
//  4. Concatenate and SHA256-hash the result.
//  5. Return the lower-case hex digest.
//
// Deterministic across OS/filesystem orderings because of the sort step.
func stagedTreeDigest(dir string) (string, error) {
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
		if rel == "manifest.json" {
			return nil // exclude manifest from the digest
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("stagedTreeDigest: open %s: %w", rel, err)
		}
		defer f.Close()

		content, err := io.ReadAll(f)
		if err != nil {
			return fmt.Errorf("stagedTreeDigest: read %s: %w", rel, err)
		}
		entries = append(entries, rel+"\x00"+string(content))
		return nil
	})
	if err != nil {
		return "", err
	}

	sort.Strings(entries)
	var combined string
	for _, e := range entries {
		combined += e
	}
	sum := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(sum[:]), nil
}
