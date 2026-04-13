package plugins

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGithubResolver_Scheme(t *testing.T) {
	if NewGithubResolver().Scheme() != "github" {
		t.Error("scheme must be 'github'")
	}
}

// Stub git runner that writes a synthetic repo tree into the clone
// target dir, so tests don't need a real git binary or network.
func stubGit(repoContents map[string]string) func(ctx context.Context, dir string, args ...string) error {
	return func(ctx context.Context, dir string, args ...string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if len(args) < 2 || args[0] != "clone" {
			return errors.New("unexpected git args")
		}
		target := args[len(args)-1]
		if err := os.MkdirAll(target, 0o755); err != nil {
			return err
		}
		// Synthesize a .git dir so we can prove the resolver strips it.
		if err := os.MkdirAll(filepath.Join(target, ".git"), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(target, ".git", "HEAD"), []byte("ref: refs/heads/main"), 0o644); err != nil {
			return err
		}
		for path, content := range repoContents {
			full := filepath.Join(target, path)
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
				return err
			}
		}
		return nil
	}
}

func TestGithubResolver_ClonesAndStripsGitDir(t *testing.T) {
	r := &GithubResolver{
		GitRunner: stubGit(map[string]string{
			"plugin.yaml":             "name: demo\n",
			"skills/h/SKILL.md":       "---\nname: h\ndescription: d\n---\n",
			"adapters/claude_code.py": "from plugins_registry.builtins import AgentskillsAdaptor as Adaptor\n",
		}),
		BaseURL: "file:///dev/null",
	}
	dst := t.TempDir()
	name, err := r.Fetch(context.Background(), "org/repo", dst)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if name != "repo" {
		t.Errorf("got name %q, want 'repo'", name)
	}
	// Contents copied.
	for _, want := range []string{"plugin.yaml", "skills/h/SKILL.md", "adapters/claude_code.py"} {
		if _, err := os.Stat(filepath.Join(dst, want)); err != nil {
			t.Errorf("missing %q: %v", want, err)
		}
	}
	// .git was stripped from the clone target before copy, so dst has no .git.
	if _, err := os.Stat(filepath.Join(dst, ".git")); !os.IsNotExist(err) {
		t.Error(".git dir must not survive into dst")
	}
}

func TestGithubResolver_PassesRefAsBranch(t *testing.T) {
	var seenArgs []string
	r := &GithubResolver{
		GitRunner: func(ctx context.Context, dir string, args ...string) error {
			seenArgs = args
			target := args[len(args)-1]
			_ = os.MkdirAll(target, 0o755)
			return nil
		},
	}
	if _, err := r.Fetch(context.Background(), "org/repo#v1.2.0", t.TempDir()); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !containsArg(seenArgs, "--branch") || !containsArg(seenArgs, "v1.2.0") {
		t.Errorf("args should include --branch v1.2.0, got %v", seenArgs)
	}
}

func TestGithubResolver_OmitsBranchFlagWhenNoRef(t *testing.T) {
	var seenArgs []string
	r := &GithubResolver{
		GitRunner: func(ctx context.Context, dir string, args ...string) error {
			seenArgs = args
			target := args[len(args)-1]
			_ = os.MkdirAll(target, 0o755)
			return nil
		},
	}
	if _, err := r.Fetch(context.Background(), "org/repo", t.TempDir()); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if containsArg(seenArgs, "--branch") {
		t.Errorf("no ref → no --branch flag, got %v", seenArgs)
	}
}

func TestGithubResolver_RejectsInvalidSpec(t *testing.T) {
	r := NewGithubResolver()
	for _, spec := range []string{
		"",
		"single-segment",
		"too/many/segments",
		"/leading-slash",
		"trailing/",
		"bad char/repo",
		"org/repo#bad ref",
	} {
		t.Run(spec, func(t *testing.T) {
			_, err := r.Fetch(context.Background(), spec, t.TempDir())
			if err == nil {
				t.Errorf("should have rejected %q", spec)
			}
		})
	}
}

func TestGithubResolver_BubblesUpGitError(t *testing.T) {
	r := &GithubResolver{
		GitRunner: func(ctx context.Context, dir string, args ...string) error {
			return errors.New("simulated auth failure")
		},
	}
	_, err := r.Fetch(context.Background(), "org/repo", t.TempDir())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "simulated") {
		t.Errorf("error should bubble git failure: %v", err)
	}
}

func TestGithubResolver_UsesDefaultsWhenNilFields(t *testing.T) {
	// A zero-value GithubResolver should still have defaults filled in
	// at Fetch time. Verified indirectly: we pass a stub that records
	// the URL passed to `git clone`.
	var seenArgs []string
	r := &GithubResolver{}
	r.GitRunner = func(ctx context.Context, dir string, args ...string) error {
		seenArgs = args
		target := args[len(args)-1]
		return os.MkdirAll(target, 0o755)
	}
	if _, err := r.Fetch(context.Background(), "org/repo", t.TempDir()); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	url := ""
	for _, a := range seenArgs {
		if strings.HasPrefix(a, "http") {
			url = a
			break
		}
	}
	if !strings.HasPrefix(url, "https://github.com/org/repo") {
		t.Errorf("default BaseURL not applied, got %q", url)
	}
}

func containsArg(args []string, target string) bool {
	for _, a := range args {
		if a == target {
			return true
		}
	}
	return false
}

// ---- defaultGitRunner ----

func TestDefaultGitRunner_PropagatesFailureFromMissingGit(t *testing.T) {
	t.Setenv("PATH", "/nonexistent")
	err := defaultGitRunner(context.Background(), t.TempDir(), "status")
	if err == nil {
		t.Error("expected error when git is unavailable on PATH")
	}
}

func TestDefaultGitRunner_UsesWorkingDirHomeFallback(t *testing.T) {
	// Force HOME empty so the resolver adds the fallback.
	t.Setenv("HOME", "")
	// Still need real git or a bogus arg. Use `--version` which succeeds
	// on any system that has git, then skip if not.
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed on this system")
	}
	if err := defaultGitRunner(context.Background(), t.TempDir(), "--version"); err != nil {
		t.Errorf("git --version should succeed: %v", err)
	}
}

func TestGithubResolver_NilGitRunnerUsesDefault(t *testing.T) {
	// Passing nil GitRunner should fall back to defaultGitRunner. With no
	// git on PATH, that fallback errors — we don't need real git here.
	t.Setenv("PATH", "/nonexistent")
	r := &GithubResolver{GitRunner: nil, BaseURL: "https://example.com"}
	_, err := r.Fetch(context.Background(), "org/repo", t.TempDir())
	if err == nil {
		t.Error("expected default git runner to error without a git binary")
	}
}

func TestGithubResolver_CopyToDstFailure(t *testing.T) {
	r := &GithubResolver{
		GitRunner: stubGit(map[string]string{"plugin.yaml": "name: x\n"}),
	}
	// Make dst read-only so copyTree fails after the successful clone.
	dst := t.TempDir()
	if err := os.Chmod(dst, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dst, 0o755) })
	_, err := r.Fetch(context.Background(), "org/repo", dst)
	if err == nil {
		t.Error("expected copy failure when dst is read-only")
	}
}

func TestGithubResolver_AlwaysPassesDepth1(t *testing.T) {
	var seenArgs []string
	r := &GithubResolver{
		GitRunner: func(ctx context.Context, dir string, args ...string) error {
			seenArgs = args
			target := args[len(args)-1]
			return os.MkdirAll(target, 0o755)
		},
	}
	if _, err := r.Fetch(context.Background(), "org/repo", t.TempDir()); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !containsArg(seenArgs, "--depth=1") {
		t.Errorf("expected --depth=1 in git args, got %v", seenArgs)
	}
}

func TestGithubResolver_PassesDoubleDashBeforeURL(t *testing.T) {
	// When a ref is specified, we pass `--` after --branch <ref> as
	// defense-in-depth against ref-as-flag injection.
	var seenArgs []string
	r := &GithubResolver{
		GitRunner: func(ctx context.Context, dir string, args ...string) error {
			seenArgs = args
			target := args[len(args)-1]
			return os.MkdirAll(target, 0o755)
		},
	}
	if _, err := r.Fetch(context.Background(), "org/repo#main", t.TempDir()); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !containsArg(seenArgs, "--") {
		t.Errorf("expected `--` separator in git args, got %v", seenArgs)
	}
}

func TestGithubResolver_RejectsRefStartingWithHyphen(t *testing.T) {
	r := NewGithubResolver()
	_, err := r.Fetch(context.Background(), "org/repo#-exec=/evil", t.TempDir())
	if err == nil {
		t.Error("ref starting with '-' must be rejected")
	}
}

func TestGithubResolver_MapsRepositoryNotFoundToSentinel(t *testing.T) {
	r := &GithubResolver{
		GitRunner: func(ctx context.Context, dir string, args ...string) error {
			return errors.New("remote: Repository not found.\nfatal: repository 'https://github.com/x/y.git' not found")
		},
	}
	_, err := r.Fetch(context.Background(), "org/repo", t.TempDir())
	if !errors.Is(err, ErrPluginNotFound) {
		t.Errorf("expected ErrPluginNotFound, got %v", err)
	}
}

func TestGithubResolver_MapsMissingBranchToSentinel(t *testing.T) {
	r := &GithubResolver{
		GitRunner: func(ctx context.Context, dir string, args ...string) error {
			return errors.New("fatal: Remote branch bogus not found in upstream origin")
		},
	}
	_, err := r.Fetch(context.Background(), "org/repo#bogus", t.TempDir())
	if !errors.Is(err, ErrPluginNotFound) {
		t.Errorf("expected ErrPluginNotFound for missing ref, got %v", err)
	}
}

func TestGithubResolver_AuthFailureIsNotErrPluginNotFound(t *testing.T) {
	r := &GithubResolver{
		GitRunner: func(ctx context.Context, dir string, args ...string) error {
			return errors.New("fatal: Authentication failed for 'https://github.com/private/repo.git/'")
		},
	}
	_, err := r.Fetch(context.Background(), "private/repo", t.TempDir())
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, ErrPluginNotFound) {
		t.Errorf("auth failure must not surface as ErrPluginNotFound: %v", err)
	}
}
