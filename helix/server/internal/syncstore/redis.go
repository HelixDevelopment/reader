// Package syncstore stores deterministic offline-first reading progress.
package syncstore

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	coreSync "github.com/helixreader/helix/core/pkg/sync"
	"github.com/redis/go-redis/v9"
)

var ErrNotFound = errors.New("progress not found")

type Store interface {
	Get(context.Context, string, string) (coreSync.ProgressRecord, error)
	Put(context.Context, string, coreSync.ProgressRecord) (coreSync.ProgressRecord, error)
	Delete(context.Context, string, string) error
	Close() error
}
type Redis struct {
	client *redis.Client
	Prefix string
}

func NewRedis(client *redis.Client) *Redis {
	return &Redis{client: client, Prefix: "helix:v1:progress:"}
}
func (r *Redis) Close() error                 { return r.client.Close() }
func part(value string) string                { return base64.RawURLEncoding.EncodeToString([]byte(value)) }
func (r *Redis) key(user, book string) string { return r.Prefix + part(user) + ":" + part(book) }
func validate(record coreSync.ProgressRecord) error {
	if record.BookID == "" || record.DeviceID == "" || record.Clock == nil {
		return fmt.Errorf("progress: book, device and clock required")
	}
	if record.Pos.Chapter < 0 || record.Pos.Block < 0 || record.Pos.Offset < 0 {
		return fmt.Errorf("progress: negative position")
	}
	if len(record.Clock.V) > 1024 {
		return fmt.Errorf("progress: too many clock entries")
	}
	return nil
}
func decode(data string) (coreSync.ProgressRecord, error) {
	var record coreSync.ProgressRecord
	if err := json.Unmarshal([]byte(data), &record); err != nil {
		return record, err
	}
	if record.Clock == nil {
		record.Clock = coreSync.New()
	}
	return record, nil
}
func (r *Redis) Get(ctx context.Context, user, book string) (coreSync.ProgressRecord, error) {
	data, err := r.client.Get(ctx, r.key(user, book)).Result()
	if errors.Is(err, redis.Nil) {
		return coreSync.ProgressRecord{}, ErrNotFound
	}
	if err != nil {
		return coreSync.ProgressRecord{}, err
	}
	return decode(data)
}
func (r *Redis) Put(ctx context.Context, user string, incoming coreSync.ProgressRecord) (coreSync.ProgressRecord, error) {
	if user == "" {
		return incoming, fmt.Errorf("progress: user required")
	}
	if err := validate(incoming); err != nil {
		return incoming, err
	}
	key := r.key(user, incoming.BookID)
	for attempts := 0; attempts < 16; attempts++ {
		var resolved coreSync.ProgressRecord
		err := r.client.Watch(ctx, func(tx *redis.Tx) error {
			data, err := tx.Get(ctx, key).Result()
			if err == nil {
				existing, err := decode(data)
				if err != nil {
					return err
				}
				resolved, err = coreSync.ResolveProgress(existing, incoming)
				if err != nil {
					return err
				}
			} else if errors.Is(err, redis.Nil) {
				resolved = incoming
				resolved.Clock = incoming.Clock.Clone()
			} else {
				return err
			}
			payload, err := json.Marshal(resolved)
			if err != nil {
				return err
			}
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error { pipe.Set(ctx, key, payload, 0); return nil })
			return err
		}, key)
		if err == nil {
			return resolved, nil
		}
		if !errors.Is(err, redis.TxFailedErr) {
			return incoming, err
		}
	}
	return incoming, fmt.Errorf("progress: concurrent update retry limit reached")
}
func (r *Redis) Delete(ctx context.Context, user, book string) error {
	return r.client.Del(ctx, r.key(user, book)).Err()
}
