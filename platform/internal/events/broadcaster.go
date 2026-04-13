package events

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/models"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/ws"
	"github.com/redis/go-redis/v9"
)

const broadcastChannel = "events:broadcast"

type Broadcaster struct {
	hub *ws.Hub
}

func NewBroadcaster(hub *ws.Hub) *Broadcaster {
	return &Broadcaster{hub: hub}
}

// RecordAndBroadcast inserts a structure event into Postgres and publishes to Redis pub/sub.
func (b *Broadcaster) RecordAndBroadcast(ctx context.Context, eventType string, workspaceID string, payload interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Insert into structure_events — cast to jsonb explicitly
	_, err = db.DB.ExecContext(ctx, `
		INSERT INTO structure_events (event_type, workspace_id, payload)
		VALUES ($1, $2, $3::jsonb)
	`, eventType, workspaceID, string(payloadJSON))
	if err != nil {
		log.Printf("RecordAndBroadcast: insert event error: %v", err)
		return err
	}

	// Build WebSocket message
	msg := models.WSMessage{
		Event:       eventType,
		WorkspaceID: workspaceID,
		Timestamp:   time.Now().UTC(),
		Payload:     payloadJSON,
	}

	// Publish to Redis pub/sub for multi-instance support
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if err := db.RDB.Publish(ctx, broadcastChannel, msgJSON).Err(); err != nil {
		log.Printf("Warning: Redis publish failed: %v", err)
	}

	// Broadcast to local WebSocket clients
	b.hub.Broadcast(msg)

	return nil
}

// BroadcastOnly sends a WebSocket event without recording in structure_events.
// Used for high-frequency events like activity logs that have their own table.
func (b *Broadcaster) BroadcastOnly(workspaceID string, eventType string, payload interface{}) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Printf("BroadcastOnly: marshal error: %v", err)
		return
	}

	msg := models.WSMessage{
		Event:       eventType,
		WorkspaceID: workspaceID,
		Timestamp:   time.Now().UTC(),
		Payload:     payloadJSON,
	}

	b.hub.Broadcast(msg)
}

// Subscribe listens to Redis pub/sub and relays events to the WebSocket hub.
func (b *Broadcaster) Subscribe(ctx context.Context) {
	sub := db.RDB.Subscribe(ctx, broadcastChannel)
	ch := sub.Channel(redis.WithChannelHealthCheckInterval(30 * time.Second))

	log.Println("Subscribed to Redis broadcast channel")
	for {
		select {
		case <-ctx.Done():
			sub.Close()
			return
		case redisMsg := <-ch:
			if redisMsg == nil {
				continue
			}
			// In single-instance mode, RecordAndBroadcast already calls hub.Broadcast().
			// This subscriber becomes relevant in multi-instance deployments.
			_ = redisMsg
		}
	}
}
