package handlers

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// maxIncludeDepth caps !include recursion to prevent runaway chains or
// cycles that slip past the visited-set check (e.g. relative paths that
// normalize differently on different OSes). A depth of 16 easily covers
// any realistic team/role hierarchy.
const maxIncludeDepth = 16

// resolveYAMLIncludes expands `!include <path>` directives in a YAML
// document. Used by POST /org/import + GET /org/templates to support
// splitting a single large org.yaml into per-team or per-role files.
//
// Semantics:
//   - A scalar node tagged `!include` with a string value is replaced by
//     the parsed content of the referenced file.
//   - Paths are resolved relative to `baseDir` and must stay inside it
//     (same traversal defense as resolveInsideRoot).
//   - Includes may be nested (a team file can !include a role file).
//     Cycles are detected via a visited set keyed on absolute path;
//     `maxIncludeDepth` caps total recursion depth as a belt-and-braces
//     check.
//   - Missing files return an error — fail loud during import, not at
//     runtime.
//
// Returns the expanded YAML as bytes (to keep the caller's existing
// `yaml.Unmarshal(data, ...)` flow unchanged).
func resolveYAMLIncludes(data []byte, baseDir string) ([]byte, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}

	visited := map[string]bool{}
	if err := expandNode(&root, baseDir, visited, 0); err != nil {
		return nil, err
	}

	out, err := yaml.Marshal(&root)
	if err != nil {
		return nil, fmt.Errorf("marshal expanded yaml: %w", err)
	}
	return out, nil
}

// expandNode walks the yaml.Node tree in-place and replaces any
// `!include`-tagged scalar with the parsed content of the referenced
// file. Compound nodes (document / mapping / sequence) recurse; alias
// nodes are left alone (the yaml parser already resolves them pre-tag).
func expandNode(n *yaml.Node, baseDir string, visited map[string]bool, depth int) error {
	if n == nil {
		return nil
	}
	if depth > maxIncludeDepth {
		return fmt.Errorf("!include: max depth %d exceeded (possible cycle)", maxIncludeDepth)
	}

	if n.Kind == yaml.ScalarNode && n.Tag == "!include" {
		return resolveIncludeScalar(n, baseDir, visited, depth)
	}

	for _, child := range n.Content {
		if err := expandNode(child, baseDir, visited, depth); err != nil {
			return err
		}
	}
	return nil
}

// resolveIncludeScalar replaces an `!include <path>` scalar with the
// parsed content of the referenced file. The replacement happens by
// mutating *n to take on the included file's root kind/content/tag.
func resolveIncludeScalar(n *yaml.Node, baseDir string, visited map[string]bool, depth int) error {
	rel := n.Value
	if rel == "" {
		return fmt.Errorf("!include at line %d: empty path", n.Line)
	}
	if baseDir == "" {
		return fmt.Errorf("!include %q at line %d requires a dir-based org template (no baseDir in inline-template mode)", rel, n.Line)
	}
	abs, err := resolveInsideRoot(baseDir, rel)
	if err != nil {
		return fmt.Errorf("!include %q at line %d: %w", rel, n.Line, err)
	}
	if visited[abs] {
		return fmt.Errorf("!include cycle detected at %q (line %d)", rel, n.Line)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return fmt.Errorf("!include %q at line %d: %w", rel, n.Line, err)
	}

	var sub yaml.Node
	if err := yaml.Unmarshal(data, &sub); err != nil {
		return fmt.Errorf("!include %q: parse: %w", rel, err)
	}
	// yaml.Unmarshal of a full file yields a DocumentNode wrapping the
	// actual root. Peel one layer so the includer sees the real content.
	root := &sub
	if root.Kind == yaml.DocumentNode && len(root.Content) == 1 {
		root = root.Content[0]
	}

	// Mark visited for the whole descent through this file, then recurse
	// so nested !includes inside the included file resolve too. Each file
	// gets its own baseDir (the directory containing it) so paths like
	// `!include role-a/initial.yaml` inside `teams/dev.yaml` resolve
	// relative to the team file's directory.
	visited[abs] = true
	defer delete(visited, abs)

	subDir := filepath.Dir(abs)
	if err := expandNode(root, subDir, visited, depth+1); err != nil {
		return err
	}

	// Replace the !include scalar with the resolved content in-place.
	*n = *root
	// Clear the !include tag (root's Tag is whatever kind it actually is —
	// !!map / !!seq / !!str — after unmarshal, which is correct).
	// If somehow root.Tag is still !include (shouldn't happen), drop it.
	if n.Tag == "!include" {
		n.Tag = ""
	}
	return nil
}
