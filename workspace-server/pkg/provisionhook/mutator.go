// Package provisionhook is the public extension point that lets external
// plugins mutate the env map a workspace container will boot with, just
// before the provisioner calls Start(cfg).
//
// The package lives under pkg/ (not internal/) because plugins import it
// from outside this Go module. Anything outside pkg/ is core-only.
//
// # Why this exists
//
// Auth providers (GitHub App tokens, GitLab tokens, Bitbucket app
// passwords, internal PAT vaults), secret managers (Vault, AWS Secrets
// Manager, GCP Secret Manager), per-tenant config injectors, and
// observability sidecars all want to write env vars into the workspace
// container before it starts. Each is an OPTIONAL concern that only some
// deployments need. Hardcoding any of them in the platform binary
// violates the "core stays small, capabilities are plugins" principle
// (CEO 2026-04-16, after the monorepo → 44 sub-repos split).
//
// # Plugin shape
//
// A plugin implements EnvMutator and registers an instance with a
// Registry at platform startup. The provisioner calls Run(...) on the
// registry before each workspace container starts.
//
// Plugins live in their own Go modules + repos (e.g.
// github.com/Molecule-AI/molecule-ai-plugin-github-app-auth). Each
// plugin ships its own cmd/server/main.go that imports core's startup
// function + registers the plugin's mutator. Operators deploy the
// plugin binary instead of core's vanilla cmd/server when they want
// the plugin's behaviour.
//
// # Failure handling
//
// MutateEnv returning a non-nil error aborts the provision (workspace
// is marked 'failed', container never starts). Plugins should fail open
// on transient external-service errors (log + return nil) so a flaky
// upstream doesn't block agent provisioning. Reserve errors for hard
// config bugs that the operator must fix.
//
// # Concurrency
//
// Registry is safe for concurrent registration + execution. MutateEnv
// implementations should be safe to call from goroutines (the
// provisioner runs each workspace's provision in its own goroutine).
package provisionhook

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// EnvMutator is implemented by plugins that want to inject env vars
// into a workspace container at provision time.
//
//   - Name returns a stable identifier for logging / metrics. Should
//     match the plugin's repo / module name (e.g. "github-app-auth").
//   - MutateEnv receives the workspace ID, the create payload, and a
//     mutable env map. It can read existing values, add new ones, or
//     overwrite as needed. Mutations are visible to subsequent
//     mutators in the chain (registration order).
type EnvMutator interface {
	Name() string
	MutateEnv(ctx context.Context, workspaceID string, env map[string]string) error
}

// TokenProvider is an optional interface that EnvMutator implementations
// may also satisfy. When a mutator implements TokenProvider the platform
// can serve GET /admin/github-installation-token, allowing long-running
// workspaces to fetch a fresh GitHub token without restarting.
//
// # Why a separate interface?
//
// EnvMutator.MutateEnv is called once at provision time and writes into
// an env map. Calling it again just to read the current token would be
// semantically wrong and potentially unsafe (the env map is a live
// workspace struct). TokenProvider cleanly separates "what do I inject
// at boot?" from "what is the live token right now?".
//
// # Plugin contract
//
// Token must return the current valid token and the time at which it
// will expire. If the plugin's internal cache is past its refresh
// threshold it must block until a new token is obtained before
// returning. Token should never return an expired token — callers rely
// on this guarantee and do not do their own expiry check.
//
// Returning a non-nil error causes the HTTP handler to respond 500 and
// log "[github] token refresh failed: <err>". The workspace will retry
// on its next credential-helper invocation.
type TokenProvider interface {
	Token(ctx context.Context) (token string, expiresAt time.Time, err error)
}

// Registry holds the ordered list of EnvMutator instances the
// provisioner runs before each workspace boot. Safe for concurrent
// registration + execution.
type Registry struct {
	mu       sync.RWMutex
	mutators []EnvMutator
}

// NewRegistry returns an empty registry. The platform creates one at
// startup; plugins call Register on it.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds a mutator to the chain. Mutators run in registration
// order. Registering the same instance twice is allowed (it'll run
// twice) — the registry doesn't dedupe; that's the caller's
// responsibility if dedup matters.
func (r *Registry) Register(m EnvMutator) {
	if m == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mutators = append(r.mutators, m)
}

// Len reports how many mutators are registered. Used by the platform's
// boot log so operators can see which extension hooks are wired.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.mutators)
}

// Names returns the names of registered mutators in registration order.
// Used by the boot log so operators can grep for which plugins are
// active.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, len(r.mutators))
	for i, m := range r.mutators {
		names[i] = m.Name()
	}
	return names
}

// FirstTokenProvider returns the first registered mutator that also
// implements TokenProvider, or nil if none do. Used to back the
// GET /admin/github-installation-token endpoint so long-running
// workspaces can refresh their GITHUB_TOKEN without a container restart.
//
// A nil registry returns nil (no provider configured).
func (r *Registry) FirstTokenProvider() TokenProvider {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, m := range r.mutators {
		if tp, ok := m.(TokenProvider); ok {
			return tp
		}
	}
	return nil
}

// Run calls every registered mutator in order. The first one to return
// a non-nil error aborts the chain — subsequent mutators do NOT run,
// and the error is returned to the caller (which marks the workspace
// failed).
//
// A nil registry is a no-op (returns nil) so the provisioner doesn't
// have to nil-check before calling.
func (r *Registry) Run(ctx context.Context, workspaceID string, env map[string]string) error {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	mutators := make([]EnvMutator, len(r.mutators))
	copy(mutators, r.mutators)
	r.mu.RUnlock()

	for _, m := range mutators {
		if err := m.MutateEnv(ctx, workspaceID, env); err != nil {
			return fmt.Errorf("provisionhook %q: %w", m.Name(), err)
		}
	}
	return nil
}
