package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// refreshEnvFromCP pulls the tenant's current config-plane env vars
// from the control plane and applies them via os.Setenv BEFORE any
// other code calls os.Getenv on them.
//
// Why:
//   - user-data on the tenant EC2 bakes env vars into `docker run` at
//     provision time. Those values are frozen. When we rotate a secret
//     on CP (e.g. PROVISION_SHARED_SECRET) there's no way to push the
//     new value into already-provisioned tenants.
//   - the Docker image auto-updater already pulls the latest workspace-
//     server image every 5 min. If THAT image knows how to refresh its
//     own env from the CP on startup, every tenant heals itself within
//     the update cycle — no ssh, no re-provision, no ops toil.
//
// Contract (paired with cp-side GET /cp/tenants/config):
//   Request:  GET {MOLECULE_CP_URL or https://api.moleculesai.app}/cp/tenants/config
//             Authorization: Bearer <ADMIN_TOKEN>
//             X-Molecule-Org-Id: <MOLECULE_ORG_ID>
//   Response: 200 {"MOLECULE_CP_SHARED_SECRET":"…","MOLECULE_CP_URL":"…", …}
//             401 on bearer mismatch or unknown org
//
// Best-effort: any failure logs and returns — main() keeps booting.
// Self-hosted deploys without MOLECULE_ORG_ID or ADMIN_TOKEN set
// short-circuit silently so this function is a no-op there.
func refreshEnvFromCP() error {
	orgID := os.Getenv("MOLECULE_ORG_ID")
	adminToken := os.Getenv("ADMIN_TOKEN")
	if orgID == "" || adminToken == "" {
		// Not a SaaS tenant (self-hosted dev or not yet provisioned).
		return nil
	}

	base := os.Getenv("MOLECULE_CP_URL")
	if base == "" {
		// Default to prod for any tenant that lost track of its CP URL
		// (e.g. older user-data that only set MOLECULE_ORG_ID).
		base = "https://api.moleculesai.app"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", base+"/cp/tenants/config", nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("X-Molecule-Org-Id", orgID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = $1 }()

	// 64 KiB cap — the CP only returns small JSON blobs here. An
	// unbounded read would be weaponizable if a compromised upstream
	// ever echoed back a gigabyte.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// 401 on first boot-after-restart is expected for tenants still
		// running under old user-data where admin_token on-disk hasn't
		// had its corresponding row seeded. Don't treat as fatal — just
		// log so operators can spot repeat offenders in logs.
		return fmt.Errorf("cp returned %d", resp.StatusCode)
	}

	var cfg map[string]string
	if err := json.Unmarshal(body, &cfg); err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	// Apply only strings; reject oversized values defensively. An
	// operator-supplied config should never exceed 4 KiB per key —
	// workspace-server env vars are URLs, hex secrets, short identifiers.
	const maxValueBytes = 4 << 10
	applied := 0
	for k, v := range cfg {
		if k == "" || len(v) > maxValueBytes {
			continue
		}
		if err := os.Setenv(k, v); err != nil {
			log.Printf("CP env refresh: setenv %s: %v", k, err)
			continue
		}
		applied++
	}
	log.Printf("CP env refresh: applied %d values from %s/cp/tenants/config", applied, base)
	return nil
}
