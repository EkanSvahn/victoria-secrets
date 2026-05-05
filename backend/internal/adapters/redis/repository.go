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
local obj = cjson.decode(val)
local views = obj["views_remaining"]
if views ~= nil then
  views = tonumber(views)
  if views <= 1 then
    redis.call('DEL', key)
  else
    obj["views_remaining"] = views - 1
    local updated = cjson.encode(obj)
    local ttl = redis.call('PTTL', key)
    if ttl and ttl > 0 then
      redis.call('PSETEX', key, ttl, updated)
    else
      redis.call('SET', key, updated)
    end
  end
end
return val
`)

type Repository struct {
	client *redis.Client
}

func New(client *redis.Client) *Repository {
	return &Repository{client: client}
}

func (r *Repository) Store(ctx context.Context, id string, payload ports.SecretRecord, ttlSeconds *int64) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	expiration := time.Duration(0)
	if ttlSeconds != nil {
		expiration = time.Duration(*ttlSeconds) * time.Second
	}
	ok, err := r.client.SetNX(ctx, id, b, expiration).Result()
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
	return decodeConsumeResult(result)
}

func decodeConsumeResult(result any) (*ports.SecretRecord, error) {
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
