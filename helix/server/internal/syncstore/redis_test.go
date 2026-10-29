package syncstore

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	coreSync "github.com/helixreader/helix/core/pkg/sync"
	"github.com/redis/go-redis/v9"
)

func TestRedisProgressConflictResolutionAndPersistence(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	store := NewRedis(client)
	defer store.Close()
	ctx := context.Background()
	a := coreSync.New()
	a.Increment("a")
	first := coreSync.ProgressRecord{BookID: "book", DeviceID: "a", Pos: coreSync.Position{Chapter: 2, Block: 1}, Clock: a}
	if _, err := store.Put(ctx, "user", first); err != nil {
		t.Fatal(err)
	}
	b := coreSync.New()
	b.Increment("b")
	behind := coreSync.ProgressRecord{BookID: "book", DeviceID: "b", Pos: coreSync.Position{Chapter: 1}, Clock: b}
	resolved, err := store.Put(ctx, "user", behind)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Pos.Chapter != 2 || resolved.Clock.V["a"] != 1 || resolved.Clock.V["b"] != 1 {
		t.Fatalf("conflict resolution wrong: %+v", resolved)
	}
	persisted, err := store.Get(ctx, "user", "book")
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Pos.Chapter != 2 {
		t.Fatalf("not persisted: %+v", persisted)
	}
	if err := store.Delete(ctx, "user", "book"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, "user", "book"); err != ErrNotFound {
		t.Fatalf("expected not found: %v", err)
	}
}
func TestProgressValidation(t *testing.T) {
	server := miniredis.RunT(t)
	store := NewRedis(redis.NewClient(&redis.Options{Addr: server.Addr()}))
	defer store.Close()
	if _, err := store.Put(context.Background(), "user", coreSync.ProgressRecord{BookID: "b", DeviceID: "d", Pos: coreSync.Position{Chapter: -1}, Clock: coreSync.New()}); err == nil {
		t.Fatal("accepted negative position")
	}
}
