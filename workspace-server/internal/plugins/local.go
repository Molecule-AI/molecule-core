package plugins

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// LocalResolver fetches plugins from a filesystem directory shipped with
// the platform (the canonical /plugins registry). This is the default
// source for bare names in the install API; most deployments point it
// at the repo's `plugins/` directory.
type LocalResolver struct {
	// BaseDir is the absolute path to the directory that contains one
	// subdirectory per available plugin (e.g. repo-root/plugins).
	BaseDir string
}

// NewLocalResolver constructs a LocalResolver pointing at baseDir.
func NewLocalResolver(baseDir string) *LocalResolver {
	return &LocalResolver{BaseDir: baseDir}
}

// Scheme returns "local".
func (r *LocalResolver) Scheme() string { return "local" }

// localNameRE constrains plugin names to safe identifiers. Matches
// validatePluginName in the handlers package; duplicated here so the
// plugins package has no reverse dependency.
//
// Length-bounded at 128 chars (1 + 127 tail). agentskills.io caps
// skill names at 64; our plugin-level names are a superset (collection
// of skills) so we allow a bit more headroom, but not unbounded.
var localNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)

// Fetch copies the plugin directory from BaseDir/<spec> into dst.
//
// `spec` is the plain plugin name (e.g. "molecule-dev"). Path-traversal
// attempts (slashes, "..", empty) are rejected.
func (r *LocalResolver) Fetch(ctx context.Context, spec string, dst string) (string, error) {
	name := strings.TrimSpace(spec)
	if name == "" {
		return "", fmt.Errorf("local resolver: empty plugin name")
	}
	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		return "", fmt.Errorf("local resolver: invalid plugin name %q", name)
	}
	if !localNameRE.MatchString(name) {
		return "", fmt.Errorf("local resolver: plugin name %q must match %s", name, localNameRE)
	}

	src := filepath.Join(r.BaseDir, name)
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("local resolver: plugin %q: %w", name, ErrPluginNotFound)
		}
		return "", fmt.Errorf("local resolver: stat %s: %w", src, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("local resolver: %q is not a directory", src)
	}

	// Copy the directory tree into dst (which the caller has created).
	if err := copyTree(ctx, src, dst); err != nil {
		return "", fmt.Errorf("local resolver: copy failed: %w", err)
	}

	return name, nil
}

// copyTree does a recursive copy honouring ctx cancellation. Avoids a
// dependency on os/exec (no need to shell out to cp).
func copyTree(ctx context.Context, src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode()&os.ModePerm)
		}
		return copyFile(path, target, info.Mode()&os.ModePerm)
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	_, err = io.Copy(out, in)
	return err
}
