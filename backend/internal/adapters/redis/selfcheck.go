package redis

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

func CheckEphemeralConfig(ctx context.Context, client *redis.Client) error {
	appendOnly, err := configGetSingle(ctx, client, "appendonly")
	if err != nil {
		return fmt.Errorf("config get appendonly failed: %w", err)
	}
	save, err := configGetSingle(ctx, client, "save")
	if err != nil {
		return fmt.Errorf("config get save failed: %w", err)
	}

	if appendOnly != "no" {
		return fmt.Errorf("redis appendonly must be 'no', got %q", appendOnly)
	}
	if save != "" {
		return fmt.Errorf("redis save must be empty for in-memory mode, got %q", save)
	}
	return nil
}

func configGetSingle(ctx context.Context, client *redis.Client, key string) (string, error) {
	cfg, err := client.ConfigGet(ctx, key).Result()
	if err != nil {
		return "", err
	}
	value, ok := cfg[key]
	if !ok {
		// Redis can return key casing variations on some versions.
		for k, v := range cfg {
			if strings.EqualFold(k, key) {
				value = v
				ok = true
				break
			}
		}
	}
	if !ok {
		return "", fmt.Errorf("missing redis config key %q", key)
	}
	return strings.TrimSpace(strings.ToLower(value)), nil
}
