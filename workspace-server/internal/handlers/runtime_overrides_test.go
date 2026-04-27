package handlers

import (
	"sync"
	"testing"
	"time"
)

func TestRuntimeOverrideCache_SetAndGet(t *testing.T) {
	c := &runtimeOverrideCache{}

	if _, ok := c.IdleTimeout("ws-a"); ok {
		t.Fatal("empty cache should not return any override")
	}

	c.SetIdleTimeout("ws-a", 10*time.Minute)
	got, ok := c.IdleTimeout("ws-a")
	if !ok || got != 10*time.Minute {
		t.Fatalf("expected 10m override; got=%v ok=%v", got, ok)
	}

	// Sibling workspace unaffected — pin against the trap where a
	// shared map without proper keying would leak overrides across
	// workspaces (a hard-to-debug "claude-code's longer timeout
	// somehow applied to langgraph too").
	if _, ok := c.IdleTimeout("ws-b"); ok {
		t.Fatal("override for ws-a leaked to ws-b")
	}
}

func TestRuntimeOverrideCache_ZeroOrNegativeClears(t *testing.T) {
	// Adapter dropping the override (returning None / 0 from
	// idle_timeout_override) must restore platform-default behavior.
	// If the cache held the previous value indefinitely, an adapter
	// downgrade would silently keep the longer timeout active.
	c := &runtimeOverrideCache{}
	c.SetIdleTimeout("ws-a", 10*time.Minute)
	if _, ok := c.IdleTimeout("ws-a"); !ok {
		t.Fatal("setup: override should be set")
	}

	c.SetIdleTimeout("ws-a", 0)
	if _, ok := c.IdleTimeout("ws-a"); ok {
		t.Fatal("zero duration should clear override")
	}

	c.SetIdleTimeout("ws-a", 5*time.Minute)
	c.SetIdleTimeout("ws-a", -1*time.Second)
	if _, ok := c.IdleTimeout("ws-a"); ok {
		t.Fatal("negative duration should clear override")
	}
}

func TestRuntimeOverrideCache_EmptyWorkspaceIDIgnored(t *testing.T) {
	// Defensive: a misrouted heartbeat with empty workspace_id
	// should NOT pollute the cache with a "" key. workspaceID == ""
	// is also the value dispatchA2A passes when the workspace is
	// indeterminate, and that path must not surface a stored value.
	c := &runtimeOverrideCache{}
	c.SetIdleTimeout("", 10*time.Minute)
	if _, ok := c.IdleTimeout(""); ok {
		t.Fatal("empty workspace_id must not store overrides")
	}
}

func TestRuntimeOverrideCache_SetReplaces(t *testing.T) {
	// A heartbeat with a new override value replaces, doesn't append.
	c := &runtimeOverrideCache{}
	c.SetIdleTimeout("ws-a", 10*time.Minute)
	c.SetIdleTimeout("ws-a", 20*time.Minute)
	got, _ := c.IdleTimeout("ws-a")
	if got != 20*time.Minute {
		t.Fatalf("expected 20m after replacement; got %v", got)
	}
}

func TestRuntimeOverrideCache_Reset(t *testing.T) {
	c := &runtimeOverrideCache{}
	c.SetIdleTimeout("ws-a", 10*time.Minute)
	c.SetIdleTimeout("ws-b", 20*time.Minute)
	c.Reset()
	if _, ok := c.IdleTimeout("ws-a"); ok {
		t.Fatal("reset should clear ws-a")
	}
	if _, ok := c.IdleTimeout("ws-b"); ok {
		t.Fatal("reset should clear ws-b")
	}
}

func TestRuntimeOverrideCache_ConcurrentSafe(t *testing.T) {
	// dispatchA2A reads the cache on every request; heartbeat handlers
	// write on every 30s. Different workspaces will be hot in different
	// goroutines. The sync.Map underlying the cache promises this; the
	// test pins it so a future "let me just use a regular map with a
	// mutex" change can't silently regress under load.
	c := &runtimeOverrideCache{}
	var wg sync.WaitGroup
	const N = 100

	for i := 0; i < N; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			c.SetIdleTimeout("ws", time.Duration(i+1)*time.Second)
		}(i)
		go func() {
			defer wg.Done()
			_, _ = c.IdleTimeout("ws")
		}()
	}
	wg.Wait()
	// Final value must be SOME positive duration written by one of the
	// goroutines — not corrupted, not zero.
	got, ok := c.IdleTimeout("ws")
	if !ok || got <= 0 || got > time.Duration(N)*time.Second {
		t.Fatalf("expected a valid override after concurrent writes; got %v ok=%v", got, ok)
	}
}
