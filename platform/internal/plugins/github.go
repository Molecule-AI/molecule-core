package plugins

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// GithubResolver fetches plugins from a GitHub repository.
//
// Spec format: "<owner>/<repo>#<40-char-commit-sha>"
//   - "foo/bar#abc1234...def" → fetch the specific immutable commit
//
// Branch names and tags are rejected by default because they are mutable
// (tags can be force-moved with `git tag -f`; branches can be force-pushed).
// Only a full 40-character hex commit SHA guarantees that the fetched content
// matches exactly what was audited.
//
// PLUGIN_ALLOW_UNPINNED=true (operator-controlled platform env var, never
// settable by workspace agents) lifts the SHA requirement for local dev / CI:
//   - "foo/bar"         → clone default-branch tip
//   - "foo/bar#main"    → clone at branch main
//   - "foo/bar#v1.2.0"  → clone at (movable) tag v1.2.0
//
// The resolver shells out to the `git` binary; the platform's Dockerfile
// installs git for this reason. A mockable GitRunner lets tests inject a
// fake without requiring git on the test host.
type GithubResolver struct {
	// GitRunner runs git commands. Defaults to shelling out to the
	// system `git`. Overridable in tests.
	GitRunner func(ctx context.Context, dir string, args ...string) error

	// BaseURL defaults to https://github.com. Tests point it at a local
	// file:// bare repo.
	BaseURL string
}

// NewGithubResolver constructs a resolver with sensible defaults.
func NewGithubResolver() *GithubResolver {
	return &GithubResolver{
		GitRunner: defaultGitRunner,
		BaseURL:   "https://github.com",
	}
}

// Scheme returns "github".
func (r *GithubResolver) Scheme() string { return "github" }

// repoRE matches "<owner>/<repo>" with optional "#<ref>" suffix.
//
//   - Owner / repo: must start with alphanumeric, then 0–99 chars from
//     [a-zA-Z0-9_.-]. Matches GitHub's validation.
//   - Ref: must NOT start with `-` (prevents ref-as-flag injection like
//     "-exec=/evil"). Then 0–254 chars from [a-zA-Z0-9_./-]. Disallows
//     whitespace and shell metacharacters.
var repoRE = regexp.MustCompile(
	`^([a-zA-Z0-9][a-zA-Z0-9_.\-]{0,99})/([a-zA-Z0-9][a-zA-Z0-9_.\-]{0,99})(?:#([a-zA-Z0-9_.][a-zA-Z0-9_./\-]{0,254}))?$`,
)

// shaRE matches a full 40-character lowercase hex commit SHA — the only ref
// format guaranteed to be immutable on any Git host. Branch names and tags
// are not matched; they may contain uppercase letters, dots, slashes, dashes,
// and are always shorter or longer than exactly 40 hex chars.
var shaRE = regexp.MustCompile(`^[0-9a-f]{40}$`)

// Fetch fetches the repository at the given spec and copies its contents
// (minus .git) into dst. Returns the repository name as the plugin name.
func (r *GithubResolver) Fetch(ctx context.Context, spec string, dst string) (string, error) {
	spec = strings.TrimSpace(spec)
	m := repoRE.FindStringSubmatch(spec)
	if m == nil {
		return "", fmt.Errorf("github resolver: spec %q must be <owner>/<repo>[#<ref>]", spec)
	}
	owner, repo, ref := m[1], m[2], m[3]

	// Pinned-ref enforcement (supply-chain hardening, issue #768 / VULN-004).
	//
	// Two-level gate when PLUGIN_ALLOW_UNPINNED != "true":
	//
	//   Level 1 — a bare spec (no #ref) is always rejected. The default-branch
	//              tip is mutable: its content can change silently between the
	//              time a plugin was audited and the time it is actually installed.
	//
	//   Level 2 — a ref that is NOT a full 40-character hex commit SHA is
	//              rejected. Branch names and tags are mutable:
	//                - branches can be force-pushed (git push --force)
	//                - tags   can be force-moved  (git tag -f v1.2.3 <new-sha>)
	//              Only a 40-char hex commit SHA is immutable on GitHub.
	//
	// PLUGIN_ALLOW_UNPINNED is a platform-process env var (set by the operator
	// via Fly.io secrets / Docker compose). It is NOT configurable per-workspace:
	// workspace env vars are passed to the workspace container, never back to the
	// platform process, so no agent can self-grant this bypass.
	if os.Getenv("PLUGIN_ALLOW_UNPINNED") != "true" {
		if ref == "" {
			return "", fmt.Errorf(
				"github resolver: spec %q requires a pinned commit SHA "+
					"(e.g. \"github://owner/repo#<40-char-sha>\"); "+
					"set PLUGIN_ALLOW_UNPINNED=true to override",
				spec,
			)
		}
		if !shaRE.MatchString(ref) {
			return "", fmt.Errorf(
				"github resolver: ref %q is not a pinned commit SHA — "+
					"branch names and movable tags are rejected by the supply-chain gate "+
					"(VULN-004); use a full 40-character lowercase hex SHA or set "+
					"PLUGIN_ALLOW_UNPINNED=true to override",
				ref,
			)
		}
	}

	runner := r.GitRunner
	if runner == nil {
		runner = defaultGitRunner
	}
	base := r.BaseURL
	if base == "" {
		base = "https://github.com"
	}
	url := fmt.Sprintf("%s/%s/%s.git", base, owner, repo)

	// Clone into a sibling temp dir, then move contents to dst minus .git.
	// We use a sibling (not dst itself) because `git clone` wants to create
	// the target; dst may already exist as an empty dir.
	workDir, err := os.MkdirTemp("", "molecule-gh-clone-*")
	if err != nil {
		return "", fmt.Errorf("github resolver: tempdir: %w", err)
	}
	defer os.RemoveAll(workDir)

	cloneTarget := filepath.Join(workDir, "repo")

	if shaRE.MatchString(ref) {
		// Commit SHA: `git clone --branch <sha>` is not valid — git's --branch
		// flag only accepts named refs (branches, tags). Use the fetch-by-SHA
		// protocol instead, which GitHub supports for any reachable commit:
		//
		//   git init <target>
		//   git -C <target> fetch --depth=1 <url> <sha>
		//   git -C <target> checkout FETCH_HEAD
		//
		// GitHub (and GitHub Enterprise ≥ 2.25) advertises arbitrary commit SHAs
		// as uploadable pack-line refs, so this succeeds for any commit that has
		// ever been pushed to the remote, regardless of which branch it's on.
		if err := runner(ctx, workDir, "init", "--", cloneTarget); err != nil {
			return "", fmt.Errorf("github resolver: git init: %w", err)
		}
		if err := runner(ctx, cloneTarget, "fetch", "--depth=1", "--", url, ref); err != nil {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "not our ref") ||
				strings.Contains(msg, "bad object") ||
				strings.Contains(msg, "couldn't find remote ref") {
				return "", fmt.Errorf("github resolver: %s@%s: %w", url, ref, ErrPluginNotFound)
			}
			return "", fmt.Errorf("github resolver: fetch %s@%s failed: %w", url, ref, err)
		}
		if err := runner(ctx, cloneTarget, "checkout", "FETCH_HEAD"); err != nil {
			return "", fmt.Errorf("github resolver: checkout FETCH_HEAD: %w", err)
		}
	} else {
		// Branch or tag ref — only reachable when PLUGIN_ALLOW_UNPINNED=true.
		// Use standard shallow clone with optional --branch.
		args := []string{"clone", "--depth=1"}
		if ref != "" {
			args = append(args, "--branch", ref)
		}
		// `--` unconditionally separates flags from positional args; URL +
		// target are positional. Defense in depth against any future arg-
		// parser quirks.
		args = append(args, "--", url, cloneTarget)
		if err := runner(ctx, workDir, args...); err != nil {
			// Map common "repository / ref doesn't exist" outputs to
			// ErrPluginNotFound so the handler returns 404. Everything else
			// stays as a 502 (network, auth, etc.).
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "repository not found") ||
				strings.Contains(msg, "could not find remote branch") ||
				strings.Contains(msg, "remote branch") && strings.Contains(msg, "not found") {
				return "", fmt.Errorf("github resolver: %s: %w", url, ErrPluginNotFound)
			}
			return "", fmt.Errorf("github resolver: clone %s failed: %w", url, err)
		}
	}

	// Strip .git so the plugin dir doesn't become a nested repo in the
	// workspace container's filesystem.
	if err := os.RemoveAll(filepath.Join(cloneTarget, ".git")); err != nil {
		return "", fmt.Errorf("github resolver: remove .git: %w", err)
	}

	// Move contents to dst.
	if err := copyTree(ctx, cloneTarget, dst); err != nil {
		return "", fmt.Errorf("github resolver: copy to dst: %w", err)
	}

	return repo, nil
}

// defaultGitRunner shells out to the system `git`. `dir` is the working
// directory for the command (nil/empty means current process cwd).
func defaultGitRunner(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	// Build a per-child env. We never mutate os.Environ()'s backing slice.
	childEnv := os.Environ()
	//  - HOME: `git clone` touches HOME for credential helpers even on
	//    anonymous HTTPS; set to work dir if the parent process has none.
	if os.Getenv("HOME") == "" && dir != "" {
		childEnv = append(childEnv, "HOME="+dir)
	}
	//  - LANG=C / LC_ALL=C: force English output so our ErrPluginNotFound
	//    mapping ("repository not found", "remote branch ... not found")
	//    doesn't silently stop working under a different locale.
	childEnv = append(childEnv, "LANG=C", "LC_ALL=C")
	cmd.Env = childEnv
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %v: %w (output: %s)", args, err, string(out))
	}
	return nil
}
