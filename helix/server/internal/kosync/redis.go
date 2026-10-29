package kosync

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/redis/go-redis/v9"
)

// Redis stores verifiers and progress in independent keys. User creation uses
// SET NX so concurrent registrations cannot replace an existing verifier.
type Redis struct {
	client *redis.Client
}

func NewRedis(client *redis.Client) *Redis { return &Redis{client: client} }

func part(value string) string { return base64.RawURLEncoding.EncodeToString([]byte(value)) }
func userKey(username string) string {
	return "helix:v1:kosync:user:" + part(username)
}
func progressKey(username, document string) string {
	return "helix:v1:kosync:progress:" + part(username) + ":" + part(document)
}

func (r *Redis) CreateUser(ctx context.Context, username, verifier string) error {
	created, err := r.client.SetNX(ctx, userKey(username), verifier, 0).Result()
	if err != nil {
		return err
	}
	if !created {
		return ErrUserExists
	}
	return nil
}

func (r *Redis) UserVerifier(ctx context.Context, username string) (string, error) {
	verifier, err := r.client.Get(ctx, userKey(username)).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrUserNotFound
	}
	return verifier, err
}

func (r *Redis) PutProgress(ctx context.Context, username string, progress Progress) error {
	payload, err := json.Marshal(progress)
	if err != nil {
		return err
	}
	key := progressKey(username, progress.Document)
	for attempt := 0; attempt < 16; attempt++ {
		err = r.client.Watch(ctx, func(tx *redis.Tx) error {
			existingPayload, getErr := tx.Get(ctx, key).Bytes()
			if getErr != nil && !errors.Is(getErr, redis.Nil) {
				return getErr
			}
			if getErr == nil {
				var existing Progress
				if err := json.Unmarshal(existingPayload, &existing); err != nil {
					return err
				}
				if existing.Percentage > progress.Percentage {
					return nil // A delayed update must not rewind the reader.
				}
			}
			// The write must run inside MULTI/EXEC: a plain tx.Set executes
			// immediately and the WATCH would never guard anything.
			_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Set(ctx, key, payload, 0)
				return nil
			})
			return err
		}, key)
		if !errors.Is(err, redis.TxFailedErr) {
			return err
		}
	}
	return errors.New("kosync: progress update retry limit reached")
}

func (r *Redis) GetProgress(ctx context.Context, username, document string) (Progress, error) {
	var progress Progress
	payload, err := r.client.Get(ctx, progressKey(username, document)).Bytes()
	if errors.Is(err, redis.Nil) {
		return progress, ErrProgressNotFound
	}
	if err != nil {
		return progress, err
	}
	if err := json.Unmarshal(payload, &progress); err != nil {
		return progress, err
	}
	return progress, nil
}

func (r *Redis) Close() error { return r.client.Close() }
