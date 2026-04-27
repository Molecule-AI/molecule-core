package handlers

import (
	"sync"
	"time"
)

// runtimeOverrides is the in-memory cache of per-workspace, adapter-
// declared overrides for cross-cutting capabilities. Populated by the
// heartbeat handler from HeartbeatPayload.RuntimeMetadata; consumed by
// dispatch paths (a2a_proxy.dispatchA2A reads IdleTimeout) before
// applying their own platform-default behavior.
//
// Why an in-memory cache and not a DB column:
//   - Heartbeats arrive every ~30s, so a fresh override propagates
//     within a heartbeat cycle of any change in adapter declarations.
//   - On platform restart the cache resets to empty until each
//     workspace's next heartbeat repopulates it. Worst-case window =
//     30s of platform-default behavior. Acceptable; nothing about
//     these overrides is correctness-critical (they tune timeouts +
//     enable native ownership of fallback features, not state).
//   - DB-roundtripping every dispatch would add latency to a hot
//     path (a2a_proxy is on every agent → agent call). The cache is
//     a sync.Map — atomic ptr load per dispatch, zero lock contention
//     under steady load.
//
// Stale entries: a workspace that goes offline never sends another
// heartbeat, but the cache entry persists until the platform restarts.
// Acceptable because dispatchA2A only consults the cache when actually
// dispatching to that workspace — a stale entry for an offline
// workspace just means "use the override that was active when it was
// last alive" (correct behavior; the workspace will get the same
// timeouts when it comes back).
//
// See workspace/adapter_base.py:idle_timeout_override and project
// memory `project_runtime_native_pluggable.md`.
var runtimeOverrides runtimeOverrideCache

type runtimeOverrideEntry struct {
	idleTimeout time.Duration // 0 means "no override; use global default"
}

type runtimeOverrideCache struct {
	m sync.Map // key: workspaceID (string), value: runtimeOverrideEntry
}

// SetIdleTimeout records the per-workspace idle-timeout override sent
// in the most recent heartbeat. d == 0 clears the override (falling
// back to the global default), so a runtime that previously declared
// an override and then dropped it cleanly returns to platform behavior.
func (c *runtimeOverrideCache) SetIdleTimeout(workspaceID string, d time.Duration) {
	if workspaceID == "" {
		return
	}
	if d <= 0 {
		c.m.Delete(workspaceID)
		return
	}
	c.m.Store(workspaceID, runtimeOverrideEntry{idleTimeout: d})
}

// IdleTimeout returns the per-workspace override and ok=true when one
// is in effect; ok=false means dispatchA2A should fall back to the
// global idleTimeoutDuration.
func (c *runtimeOverrideCache) IdleTimeout(workspaceID string) (time.Duration, bool) {
	v, ok := c.m.Load(workspaceID)
	if !ok {
		return 0, false
	}
	e, ok := v.(runtimeOverrideEntry)
	if !ok || e.idleTimeout <= 0 {
		return 0, false
	}
	return e.idleTimeout, true
}

// Reset clears the entire cache. Test-only; production code never
// needs this since heartbeats refresh entries naturally.
func (c *runtimeOverrideCache) Reset() {
	c.m.Range(func(k, _ any) bool {
		c.m.Delete(k)
		return true
	})
}
