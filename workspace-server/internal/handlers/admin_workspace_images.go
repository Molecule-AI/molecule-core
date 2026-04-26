package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	dockerimage "github.com/docker/docker/api/types/image"
	dockerclient "github.com/docker/docker/client"
	"github.com/gin-gonic/gin"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/provisioner"
)

// AdminWorkspaceImagesHandler serves POST /admin/workspace-images/refresh — the
// production-side end of the runtime CD chain. Operators (or post-publish
// automation) hit this to (1) pull the latest workspace template images from
// GHCR via the Docker SDK and (2) recreate any running ws-* containers so
// they adopt the new image. Without this, a freshly-published runtime sits
// in the registry but containers keep running the old image until the next
// manual restart.
//
// On a SaaS deployment the deploy pipeline already pulls on every release,
// so the pull step is a no-op there; the recreate step is still the way to
// make running workspaces adopt the new image without a full host restart.
//
// POST /admin/workspace-images/refresh
//
//	?runtime=claude-code   (optional; default = all 8 templates)
//	&recreate=true|false   (default true; false = pull only)
//
// Returns JSON {pulled: [...], failed: [...], recreated: [...]}
type AdminWorkspaceImagesHandler struct {
	docker *dockerclient.Client
}

func NewAdminWorkspaceImagesHandler(docker *dockerclient.Client) *AdminWorkspaceImagesHandler {
	return &AdminWorkspaceImagesHandler{docker: docker}
}

// allRuntimes is the canonical list mirroring docs/workspace-runtime-package.md.
// Update both when a new template is added.
var allRuntimes = []string{
	"claude-code", "langgraph", "crewai", "autogen",
	"deepagents", "hermes", "gemini-cli", "openclaw",
}

type refreshResult struct {
	Pulled    []string `json:"pulled"`
	Failed    []string `json:"failed"`
	Recreated []string `json:"recreated"`
}

// ghcrAuthHeader returns the base64-encoded JSON auth payload Docker's
// ImagePull expects in PullOptions.RegistryAuth, or empty string when no
// GHCR_USER/GHCR_TOKEN env is set (lets public images pull through).
//
// The Docker SDK doesn't read ~/.docker/config.json — every authenticated
// pull needs an explicit RegistryAuth string. Format per the Docker
// engine API: {"username":"…","password":"…","serveraddress":"ghcr.io"}
// → base64-encoded JSON with no trailing padding stripped (engine handles
// either form).
func ghcrAuthHeader() string {
	user := strings.TrimSpace(os.Getenv("GHCR_USER"))
	token := strings.TrimSpace(os.Getenv("GHCR_TOKEN"))
	if user == "" || token == "" {
		return ""
	}
	payload := map[string]string{
		"username":      user,
		"password":      token,
		"serveraddress": "ghcr.io",
	}
	js, err := json.Marshal(payload)
	if err != nil {
		// Should be unreachable for a static map[string]string. Log so a
		// future contributor adding a non-marshallable field notices.
		log.Printf("workspace-images: failed to marshal GHCR auth: %v", err)
		return ""
	}
	return base64.URLEncoding.EncodeToString(js)
}

func (h *AdminWorkspaceImagesHandler) Refresh(c *gin.Context) {
	runtimes := allRuntimes
	if r := c.Query("runtime"); r != "" {
		// Accept a single runtime; reject anything not in the canonical list
		// so a typo doesn't silently no-op.
		found := false
		for _, known := range allRuntimes {
			if known == r {
				found = true
				break
			}
		}
		if !found {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":          fmt.Sprintf("unknown runtime: %s", r),
				"known_runtimes": allRuntimes,
			})
			return
		}
		runtimes = []string{r}
	}
	recreate := c.DefaultQuery("recreate", "true") == "true"

	res := refreshResult{Pulled: []string{}, Failed: []string{}, Recreated: []string{}}
	auth := ghcrAuthHeader()

	// 1. Pull each template image via the Docker SDK. Soft-fail per-runtime
	//    so one missing image (e.g. unpublished template) doesn't abort
	//    the others. Each pull's progress stream is drained to completion
	//    — the engine treats early-close as "abandon", leaving partial
	//    layers around with no reference.
	pullCtx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()
	for _, rt := range runtimes {
		image := fmt.Sprintf("ghcr.io/molecule-ai/workspace-template-%s:latest", rt)
		opts := dockerimage.PullOptions{Platform: provisioner.DefaultImagePlatform()}
		if auth != "" {
			opts.RegistryAuth = auth
		}
		rc, err := h.docker.ImagePull(pullCtx, image, opts)
		if err != nil {
			log.Printf("workspace-images/refresh: pull %s failed: %v", rt, err)
			res.Failed = append(res.Failed, rt)
			continue
		}
		// Drain to completion. We discard progress payload because no
		// caller renders it; the platform log already records pulled/failed
		// per runtime. If a future caller wants live progress, decode the
		// JSON-line stream into events here.
		if _, err := io.Copy(io.Discard, rc); err != nil {
			rc.Close()
			log.Printf("workspace-images/refresh: drain %s failed: %v", rt, err)
			res.Failed = append(res.Failed, rt)
			continue
		}
		rc.Close()
		res.Pulled = append(res.Pulled, rt)
	}

	if !recreate {
		c.JSON(http.StatusOK, res)
		return
	}

	// 2. Find ws-* containers running an image we just pulled. Recreate
	//    them — kill+remove and let the platform's normal provisioning
	//    flow re-create on next canvas interaction.
	listCtx, listCancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer listCancel()
	containers, err := h.docker.ContainerList(listCtx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("name", "ws-")),
	})
	if err != nil {
		log.Printf("workspace-images/refresh: container list failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "container list failed", "partial_result": res})
		return
	}

	pulledSet := map[string]struct{}{}
	for _, rt := range res.Pulled {
		pulledSet[rt] = struct{}{}
	}
	for _, ctr := range containers {
		// ContainerList's ctr.Image is the *resolved digest* (sha256:…),
		// not the human-readable tag. Use ContainerInspect to get the
		// original Config.Image (e.g. "ghcr.io/molecule-ai/workspace-
		// template-claude-code:latest") so we can match against the
		// pulled-runtime set. The cost is one extra round-trip per
		// ws-* container — there are at most 8 typically, so this is
		// well below any UX threshold.
		inspectCtx, inspectCancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		full, err := h.docker.ContainerInspect(inspectCtx, ctr.ID)
		inspectCancel()
		if err != nil {
			log.Printf("workspace-images/refresh: inspect %s failed: %v", ctr.ID[:12], err)
			continue
		}
		imageRef := ""
		if full.Config != nil {
			imageRef = full.Config.Image
		}
		matched := ""
		for rt := range pulledSet {
			if strings.Contains(imageRef, "workspace-template-"+rt) {
				matched = rt
				break
			}
		}
		if matched == "" {
			continue
		}
		name := strings.TrimPrefix(ctr.Names[0], "/")
		// Remove with force — the workspace will re-provision on the next
		// canvas interaction. This drops in-flight conversations on the
		// removed container; document via the response so callers can
		// schedule the refresh during a quiet window.
		rmCtx, rmCancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		err = h.docker.ContainerRemove(rmCtx, ctr.ID, container.RemoveOptions{Force: true})
		rmCancel()
		if err != nil {
			log.Printf("workspace-images/refresh: remove %s failed: %v", name, err)
			continue
		}
		res.Recreated = append(res.Recreated, name)
	}

	authStatus := "no GHCR auth (public images only)"
	if auth != "" {
		authStatus = "GHCR_USER/GHCR_TOKEN auth"
	}
	log.Printf("workspace-images/refresh: pulled=%d failed=%d recreated=%d (%s)",
		len(res.Pulled), len(res.Failed), len(res.Recreated), authStatus)
	c.JSON(http.StatusOK, res)
}
