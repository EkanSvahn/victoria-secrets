package ports

import "context"

type SecretRecord struct {
	Meta       string `json:"meta"`
	Ciphertext string `json:"ciphertext"`
	Kind       string `json:"kind"`
}

type SecretRepository interface {
	Store(ctx context.Context, id string, payload SecretRecord, ttlSeconds int64) error
	Consume(ctx context.Context, id string) (*SecretRecord, error)
	Exists(ctx context.Context, id string) (bool, error)
}
