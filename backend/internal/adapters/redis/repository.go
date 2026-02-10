package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"victora-secret-code/backend/internal/ports"
)

var consumeLua = redis.NewScript(`
local key = KEYS[1]
local val = redis.call('GET', key)
if not val then
  return nil
end
redis.call('DEL', key)
return val
`)

type Repository struct {
	client *redis.Client
}

func New(client *redis.Client) *Repository {
	return &Repository{client: client}
}

func (r *Repository) Store(ctx context.Context, id string, payload ports.SecretRecord, ttlSeconds int64) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	ok, err := r.client.SetNX(ctx, id, b, time.Duration(ttlSeconds)*time.Second).Result()
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("id already exists")
	}
	return nil
}

func (r *Repository) Consume(ctx context.Context, id string) (*ports.SecretRecord, error) {
	result, err := consumeLua.Run(ctx, r.client, []string{id}).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	data, ok := result.(string)
	if !ok {
		return nil, errors.New("unexpected payload type")
	}
	var record ports.SecretRecord
	if err := json.Unmarshal([]byte(data), &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *Repository) Exists(ctx context.Context, id string) (bool, error) {
	n, err := r.client.Exists(ctx, id).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
