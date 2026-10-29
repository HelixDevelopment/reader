package kosync

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestLiveRedisKOSyncRoundTrip(t *testing.T) {
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
	if err := client.Del(ctx, userKey("live-reader"), progressKey("live-reader", "live-document")).Err(); err != nil {
		client.Close()
		t.Fatal(err)
	}
	store := NewRedis(client)
	defer func() {
		_ = client.Del(context.Background(), userKey("live-reader"), progressKey("live-reader", "live-document")).Err()
		_ = store.Close()
	}()
	if err := store.CreateUser(ctx, "live-reader", "live-verifier"); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateUser(ctx, "live-reader", "replacement"); !errors.Is(err, ErrUserExists) {
		t.Fatalf("duplicate registration replaced verifier: %v", err)
	}
	progress := Progress{Document: "live-document", Progress: "42", Percentage: 0.25, Device: "live-device", Timestamp: 123}
	if err := store.PutProgress(ctx, "live-reader", progress); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	client = redis.NewClient(&redis.Options{Addr: addr})
	store = NewRedis(client)
	got, err := store.GetProgress(ctx, "live-reader", "live-document")
	if err != nil || got != progress {
		t.Fatalf("live Redis progress did not survive reconnect: %+v %v", got, err)
	}
	if _, err := store.GetProgress(ctx, "other-reader", "live-document"); !errors.Is(err, ErrProgressNotFound) {
		t.Fatalf("live Redis user isolation failed: %v", err)
	}
}
