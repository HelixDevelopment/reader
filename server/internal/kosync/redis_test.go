package kosync

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// conflictHook deterministically lands a newer progress write from a second
// client between the store's WATCH and EXEC: the first time the store reads
// the watched key inside the transaction callback, the hook rewrites it. The
// EXEC must then abort and the read-modify-write must retry.
type conflictHook struct {
	writer  *redis.Client
	key     string
	payload []byte
	fired   atomic.Int32
}

func (h *conflictHook) DialHook(next redis.DialHook) redis.DialHook { return next }
func (h *conflictHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func (h *conflictHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		err := next(ctx, cmd)
		// Rewrite after the read but before the transaction executes.
		if cmd.Name() == "get" && len(cmd.Args()) > 1 && cmd.Args()[1] == h.key &&
			h.fired.CompareAndSwap(0, 1) {
			if setErr := h.writer.Set(ctx, h.key, h.payload, 0).Err(); setErr != nil {
				return setErr
			}
		}
		return err
	}
}

func TestRedisPutProgressRetriesWatchConflict(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	store := NewRedis(client)
	defer store.Close()
	writer := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer writer.Close()
	ctx := context.Background()

	stored := Progress{Document: "hash", Progress: "50", Percentage: .50, Device: "kindle", Timestamp: 10}
	if err := store.PutProgress(ctx, "reader", stored); err != nil {
		t.Fatal(err)
	}
	// A racing device lands a newer position while the update below holds a
	// WATCH on the key but has not executed its transaction yet.
	newer := Progress{Document: "hash", Progress: "90", Percentage: .90, Device: "phone", Timestamp: 11}
	newerPayload, err := json.Marshal(newer)
	if err != nil {
		t.Fatal(err)
	}
	hook := &conflictHook{writer: writer, key: progressKey("reader", "hash"), payload: newerPayload}
	client.AddHook(hook)

	// This update would advance 0.50, so the first attempt tries to write;
	// after the forced conflict the retry must observe 0.90 and stand down.
	stale := Progress{Document: "hash", Progress: "70", Percentage: .70, Device: "stale", Timestamp: 12}
	if err := store.PutProgress(ctx, "reader", stale); err != nil {
		t.Fatalf("update must retry the aborted transaction: %v", err)
	}
	if got := hook.fired.Load(); got != 1 {
		t.Fatalf("expected exactly one forced conflict, got %d", got)
	}
	got, err := store.GetProgress(ctx, "reader", "hash")
	if err != nil {
		t.Fatal(err)
	}
	if got != newer {
		t.Fatalf("stale lower-percentage update rewound progress: got %+v want %+v", got, newer)
	}
}

func TestRedisStoreRoundTripAndIsolation(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	store := NewRedis(client)
	defer store.Close()
	ctx := context.Background()
	if err := store.CreateUser(ctx, "reader", "bcrypt-verifier"); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateUser(ctx, "reader", "replacement"); !errors.Is(err, ErrUserExists) {
		t.Fatalf("verifier replacement accepted: %v", err)
	}
	verifier, err := store.UserVerifier(ctx, "reader")
	if err != nil || verifier != "bcrypt-verifier" {
		t.Fatalf("verifier: %q %v", verifier, err)
	}
	progress := Progress{Document: "hash", Progress: "42", Percentage: .25, Device: "device", Timestamp: 123}
	if err := store.PutProgress(ctx, "reader", progress); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetProgress(ctx, "reader", "hash")
	if err != nil || got != progress {
		t.Fatalf("round trip: %+v %v", got, err)
	}
	stale := progress
	stale.Progress = "2"
	stale.Percentage = .02
	stale.Timestamp = 999
	if err := store.PutProgress(ctx, "reader", stale); err != nil {
		t.Fatal(err)
	}
	got, err = store.GetProgress(ctx, "reader", "hash")
	if err != nil || got != progress {
		t.Fatalf("stale Redis write rewound progress: %+v %v", got, err)
	}
	if _, err := store.GetProgress(ctx, "other", "hash"); !errors.Is(err, ErrProgressNotFound) {
		t.Fatalf("cross-user key collision: %v", err)
	}
}
