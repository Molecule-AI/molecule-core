package handlers

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// runtimeProvisionTimeouts caches the per-runtime provision-timeout values
// declared in template config.yaml manifests (#2054 phase 2). Lazy-init so
// the first workspace API request after process start pays the read cost
// (a few KB of yaml across ~50 templates) and every subsequent one is a
// map lookup.
//
// Cache lifetime = process lifetime. Templates only change on container
// rebuild + workspace-server restart, which already invalidates the
// in-memory state. A future template-hot-reload feature would need to
// refresh this cache; today there is no such hook.
type runtimeProvisionTimeoutsCache struct {
	once sync.Once
	m    map[string]int // runtime → seconds
}

func (c *runtimeProvisionTimeoutsCache) get(configsDir string, runtime string) int {
	c.once.Do(func() {
		c.m = loadRuntimeProvisionTimeouts(configsDir)
	})
	return c.m[runtime]
}

// loadRuntimeProvisionTimeouts walks `configsDir`, parses every immediate
// subdirectory's `config.yaml`, and returns a map of runtime → seconds
// for templates that declared `runtime_config.provision_timeout_seconds`.
//
// Templates without the field aren't represented (lookup returns zero,
// which the caller treats as "fall through to canvas runtime profile").
//
// Multiple templates with the same runtime: take the MAX timeout — a
// slow template's threshold should win over a fast template's so users
// of either template see a true-positive timeout signal rather than a
// false alarm. Same-runtime divergence is rare in practice (typically
// one canonical template-{runtime} per runtime) but the rule is the
// safer default.
func loadRuntimeProvisionTimeouts(configsDir string) map[string]int {
	out := map[string]int{}
	entries, err := os.ReadDir(configsDir)
	if err != nil {
		// Logged but not fatal — workspace-server starts cleanly with
		// no templates (dev / fresh-clone). The result is an empty map
		// so every runtime falls through to canvas's profile default.
		log.Printf("loadRuntimeProvisionTimeouts: read configsDir %s: %v", configsDir, err)
		return out
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(configsDir, e.Name(), "config.yaml"))
		if err != nil {
			continue
		}
		var raw struct {
			Runtime       string `yaml:"runtime"`
			RuntimeConfig struct {
				ProvisionTimeoutSeconds int `yaml:"provision_timeout_seconds"`
			} `yaml:"runtime_config"`
		}
		if err := yaml.Unmarshal(data, &raw); err != nil {
			continue
		}
		secs := raw.RuntimeConfig.ProvisionTimeoutSeconds
		if secs <= 0 || raw.Runtime == "" {
			continue
		}
		if existing, ok := out[raw.Runtime]; !ok || secs > existing {
			out[raw.Runtime] = secs
		}
	}
	return out
}
