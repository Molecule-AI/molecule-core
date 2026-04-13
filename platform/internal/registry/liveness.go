package registry

import (
	"context"
	"log"
	"strings"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
)

// OfflineHandler is called when a workspace's liveness key expires.
type OfflineHandler func(ctx context.Context, workspaceID string)

// StartLivenessMonitor subscribes to Redis keyspace expiry events.
// When a workspace's liveness key (ws:{id}) expires, it marks the workspace offline
// and calls the onOffline handler.
func StartLivenessMonitor(ctx context.Context, onOffline OfflineHandler) {
	sub := db.RDB.PSubscribe(ctx, "__keyevent@0__:expired")

	log.Println("Liveness monitor started — listening for Redis key expirations")

	ch := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			sub.Close()
			return
		case msg := <-ch:
			if msg == nil {
				continue
			}
			key := msg.Payload
			if !strings.HasPrefix(key, "ws:") {
				continue
			}
			parts := strings.SplitN(key, ":", 3)
			if len(parts) != 2 {
				continue
			}
			workspaceID := parts[1]

			log.Printf("Liveness: workspace %s TTL expired", workspaceID)

			// Mark offline in Postgres — skip paused workspaces (they have no container)
			_, err := db.DB.ExecContext(ctx, `
				UPDATE workspaces SET status = 'offline', updated_at = now()
				WHERE id = $1 AND status NOT IN ('removed', 'paused')
			`, workspaceID)
			if err != nil {
				log.Printf("Liveness: failed to mark %s offline: %v", workspaceID, err)
				continue
			}

			if onOffline != nil {
				onOffline(ctx, workspaceID)
			}
		}
	}
}
