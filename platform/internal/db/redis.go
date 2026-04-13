package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client

func InitRedis(redisURL string) error {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return fmt.Errorf("parse redis url: %w", err)
	}
	RDB = redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := RDB.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}
	log.Println("Connected to Redis")
	return nil
}

// SetOnline sets the workspace liveness key with a 60s TTL.
func SetOnline(ctx context.Context, workspaceID string) error {
	key := fmt.Sprintf("ws:%s", workspaceID)
	return RDB.Set(ctx, key, "online", 60*time.Second).Err()
}

// RefreshTTL refreshes the liveness TTL for a workspace.
func RefreshTTL(ctx context.Context, workspaceID string) error {
	key := fmt.Sprintf("ws:%s", workspaceID)
	return RDB.Expire(ctx, key, 60*time.Second).Err()
}

// CacheURL caches a workspace URL for fast resolution.
func CacheURL(ctx context.Context, workspaceID, url string) error {
	key := fmt.Sprintf("ws:%s:url", workspaceID)
	return RDB.Set(ctx, key, url, 5*time.Minute).Err()
}

// GetCachedURL gets a cached workspace URL.
func GetCachedURL(ctx context.Context, workspaceID string) (string, error) {
	key := fmt.Sprintf("ws:%s:url", workspaceID)
	return RDB.Get(ctx, key).Result()
}

// CacheInternalURL caches the Docker-internal URL for workspace-to-workspace discovery.
func CacheInternalURL(ctx context.Context, workspaceID, url string) error {
	key := fmt.Sprintf("ws:%s:internal_url", workspaceID)
	return RDB.Set(ctx, key, url, 5*time.Minute).Err()
}

// GetCachedInternalURL gets the Docker-internal URL for a workspace.
func GetCachedInternalURL(ctx context.Context, workspaceID string) (string, error) {
	key := fmt.Sprintf("ws:%s:internal_url", workspaceID)
	return RDB.Get(ctx, key).Result()
}

// ClearWorkspaceKeys removes all Redis keys for a workspace (liveness, URL cache, internal URL cache).
func ClearWorkspaceKeys(ctx context.Context, workspaceID string) {
	for _, suffix := range []string{"", ":url", ":internal_url"} {
		RDB.Del(ctx, fmt.Sprintf("ws:%s%s", workspaceID, suffix))
	}
}

// IsOnline checks if a workspace is online.
func IsOnline(ctx context.Context, workspaceID string) (bool, error) {
	key := fmt.Sprintf("ws:%s", workspaceID)
	val, err := RDB.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return val > 0, nil
}
