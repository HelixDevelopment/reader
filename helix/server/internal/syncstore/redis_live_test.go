package syncstore

import (
	"context"
	"os"
	"testing"
	"time"

	coreSync "github.com/helixreader/helix/core/pkg/sync"
	"github.com/redis/go-redis/v9"
)

func TestLiveRedisProgressConflictAndReconnect(t *testing.T) {
	addr := os.Getenv("HELIX_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("HELIX_TEST_REDIS_ADDR is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		t.Fatal(err)
	}
	store := NewRedis(client)
	_ = store.Delete(ctx, "live-user", "live-book")
	defer func() {
		_ = store.Delete(context.Background(), "live-user", "live-book")
		_ = store.Close()
	}()
	a := coreSync.New()
	a.Increment("a")
	first := coreSync.ProgressRecord{BookID: "live-book", DeviceID: "a", Pos: coreSync.Position{Chapter: 2, Block: 1}, Clock: a}
	if _, err := store.Put(ctx, "live-user", first); err != nil {
		t.Fatal(err)
	}
	b := coreSync.New()
	b.Increment("b")
	behind := coreSync.ProgressRecord{BookID: "live-book", DeviceID: "b", Pos: coreSync.Position{Chapter: 1}, Clock: b}
	resolved, err := store.Put(ctx, "live-user", behind)
	if err != nil || resolved.Pos.Chapter != 2 || resolved.Clock.V["a"] != 1 || resolved.Clock.V["b"] != 1 {
		t.Fatalf("live conflict resolution failed: %+v %v", resolved, err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	client = redis.NewClient(&redis.Options{Addr: addr})
	store = NewRedis(client)
	persisted, err := store.Get(ctx, "live-user", "live-book")
	if err != nil || persisted.Pos.Chapter != 2 {
		t.Fatalf("live progress did not survive reconnect: %+v %v", persisted, err)
	}
}
