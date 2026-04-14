package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// ListRegistry handles GET /plugins — lists all available plugins from the registry.
// Supports optional ?runtime=<name> query param to filter to plugins that
// declare support for the given runtime. Plugins with no declared
// `runtimes` field are treated as "unspecified, try it" and included.
func (h *PluginsHandler) ListRegistry(c *gin.Context) {
	runtime := c.Query("runtime")
	c.JSON(http.StatusOK, h.listRegistryFiltered(runtime))
}

// listRegistryFiltered is the shared read-plus-filter path used by both
// /plugins and /workspaces/:id/plugins/available.
func (h *PluginsHandler) listRegistryFiltered(runtime string) []pluginInfo {
	plugins := []pluginInfo{}
	entries, err := os.ReadDir(h.pluginsDir)
	if err != nil {
		return plugins
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info := h.readPluginManifest(filepath.Join(h.pluginsDir, e.Name()), e.Name())
		if runtime != "" && !info.supportsRuntime(runtime) {
			continue
		}
		plugins = append(plugins, info)
	}
	return plugins
}

// ListAvailableForWorkspace handles GET /workspaces/:id/plugins/available —
// returns plugins from the registry filtered to those supported by the
// workspace's runtime. If no runtime lookup is wired, falls back to the
// full registry.
func (h *PluginsHandler) ListAvailableForWorkspace(c *gin.Context) {
	workspaceID := c.Param("id")
	runtime := ""
	if h.runtimeLookup != nil {
		if r, err := h.runtimeLookup(workspaceID); err == nil {
			runtime = r
		}
	}
	c.JSON(http.StatusOK, h.listRegistryFiltered(runtime))
}

// ListInstalled handles GET /workspaces/:id/plugins — lists plugins installed in the workspace.
func (h *PluginsHandler) ListInstalled(c *gin.Context) {
	workspaceID := c.Param("id")
	ctx := c.Request.Context()
	plugins := []pluginInfo{}

	containerName := h.findRunningContainer(ctx, workspaceID)
	if containerName == "" {
		c.JSON(http.StatusOK, plugins)
		return
	}

	// List directories in /configs/plugins/
	output, err := h.execInContainer(ctx, containerName, []string{
		"sh", "-c", "ls -1 /configs/plugins/ 2>/dev/null || true",
	})
	if err != nil {
		c.JSON(http.StatusOK, plugins)
		return
	}

	for _, name := range strings.Split(output, "\n") {
		name = strings.TrimSpace(name)
		if name == "" || validatePluginName(name) != nil {
			continue
		}
		// Try to read manifest from container (safe: name is validated)
		manifestOutput, err := h.execInContainer(ctx, containerName, []string{
			"cat", fmt.Sprintf("/configs/plugins/%s/plugin.yaml", name),
		})
		if err != nil || manifestOutput == "" {
			plugins = append(plugins, pluginInfo{Name: name})
			continue
		}
		info := parseManifestYAML(name, []byte(manifestOutput))
		plugins = append(plugins, info)
	}

	// Annotate each installed plugin with whether it still supports the
	// workspace's current runtime. Lets the canvas grey out plugins that
	// went inert after a runtime change.
	if h.runtimeLookup != nil {
		if runtime, err := h.runtimeLookup(workspaceID); err == nil && runtime != "" {
			for i := range plugins {
				ok := plugins[i].supportsRuntime(runtime)
				plugins[i].SupportedOnRuntime = &ok
			}
		}
	}

	c.JSON(http.StatusOK, plugins)
}

// CheckRuntimeCompatibility handles GET /workspaces/:id/plugins/compatibility?runtime=<name>
// — preflight for runtime changes. Reports which installed plugins would
// become inert if the workspace switched to <runtime>. Canvas uses this
// to show a confirm dialog before applying the change.
func (h *PluginsHandler) CheckRuntimeCompatibility(c *gin.Context) {
	workspaceID := c.Param("id")
	targetRuntime := c.Query("runtime")
	ctx := c.Request.Context()

	if targetRuntime == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "runtime query parameter is required"})
		return
	}

	containerName := h.findRunningContainer(ctx, workspaceID)
	if containerName == "" {
		// Workspace not running — nothing installed yet, trivially compatible.
		c.JSON(http.StatusOK, gin.H{
			"target_runtime": targetRuntime,
			"compatible":     []pluginInfo{},
			"incompatible":   []pluginInfo{},
			"all_compatible": true,
		})
		return
	}

	output, err := h.execInContainer(ctx, containerName, []string{
		"sh", "-c", "ls -1 /configs/plugins/ 2>/dev/null || true",
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list installed plugins"})
		return
	}

	compatible := []pluginInfo{}
	incompatible := []pluginInfo{}
	for _, name := range strings.Split(output, "\n") {
		name = strings.TrimSpace(name)
		if name == "" || validatePluginName(name) != nil {
			continue
		}
		manifestOutput, err := h.execInContainer(ctx, containerName, []string{
			"cat", fmt.Sprintf("/configs/plugins/%s/plugin.yaml", name),
		})
		var info pluginInfo
		if err != nil || manifestOutput == "" {
			info = pluginInfo{Name: name}
		} else {
			info = parseManifestYAML(name, []byte(manifestOutput))
		}
		if info.supportsRuntime(targetRuntime) {
			compatible = append(compatible, info)
		} else {
			incompatible = append(incompatible, info)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"target_runtime": targetRuntime,
		"compatible":     compatible,
		"incompatible":   incompatible,
		"all_compatible": len(incompatible) == 0,
	})
}
