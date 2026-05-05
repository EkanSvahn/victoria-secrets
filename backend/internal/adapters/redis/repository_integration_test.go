//go:build integration

package redis

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"victora-secret-code/backend/internal/ports"
)

func startRedis(t *testing.T) *Repository {
	t.Helper()
	ctx := context.Background()
	container, err := tcredis.Run(ctx, "redis:7-alpine")
	if err != nil {
		t.Fatalf("start redis container: %v", err)
	}
	t.Cleanup(func() {
		testcontainers.CleanupContainer(t, container)
	})

	endpoint, err := container.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("get redis endpoint: %v", err)
	}
	client := goredis.NewClient(&goredis.Options{Addr: endpoint})
	t.Cleanup(func() { _ = client.Close() })
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping redis: %v", err)
	}
	return New(client)
}

func TestIntegrationStoreAndConsumeRoundTrip(t *testing.T) {
	repo := startRedis(t)
	ctx := context.Background()

	id := "round-trip-id"
	views := int64(1)
	record := ports.SecretRecord{
		Meta:           "m",
		Ciphertext:     "c",
		Kind:           "text",
		ViewsRemaining: &views,
	}
	if err := repo.Store(ctx, id, record, nil); err != nil {
		t.Fatalf("store: %v", err)
	}

	got, err := repo.Consume(ctx, id)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if got == nil || got.Ciphertext != "c" || got.Kind != "text" {
		t.Fatalf("unexpected consume result: %+v", got)
	}

	exists, err := repo.Exists(ctx, id)
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if exists {
		t.Fatal("expected key to be deleted after consume")
	}
}

func TestIntegrationConsumeAtomicityUnderContention(t *testing.T) {
	repo := startRedis(t)
	ctx := context.Background()

	id := "atomicity-id"
	views := int64(1)
	record := ports.SecretRecord{
		Meta:           "m",
		Ciphertext:     "c",
		Kind:           "text",
		ViewsRemaining: &views,
	}
	if err := repo.Store(ctx, id, record, nil); err != nil {
		t.Fatalf("store: %v", err)
	}

	const goroutines = 100
	var winners atomic.Int64
	var failures atomic.Int64
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			<-start
			got, err := repo.Consume(ctx, id)
			if err != nil {
				failures.Add(1)
				return
			}
			if got != nil {
				winners.Add(1)
			}
		}()
	}

	close(start)
	wg.Wait()

	if w := winners.Load(); w != 1 {
		t.Fatalf("expected exactly 1 winner under contention, got %d", w)
	}
	if f := failures.Load(); f != 0 {
		t.Fatalf("unexpected goroutine errors: %d", f)
	}

	exists, err := repo.Exists(ctx, id)
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if exists {
		t.Fatal("key must be gone after the single winner consumed it")
	}
}

func TestIntegrationTTLExpiry(t *testing.T) {
	repo := startRedis(t)
	ctx := context.Background()

	id := "ttl-id"
	record := ports.SecretRecord{Meta: "m", Ciphertext: "c", Kind: "text"}
	ttl := int64(1)
	if err := repo.Store(ctx, id, record, &ttl); err != nil {
		t.Fatalf("store: %v", err)
	}

	exists, err := repo.Exists(ctx, id)
	if err != nil {
		t.Fatalf("exists before expiry: %v", err)
	}
	if !exists {
		t.Fatal("key must exist before TTL expires")
	}

	time.Sleep(1500 * time.Millisecond)

	exists, err = repo.Exists(ctx, id)
	if err != nil {
		t.Fatalf("exists after expiry: %v", err)
	}
	if exists {
		t.Fatal("key must be gone after TTL expires")
	}
}

func TestIntegrationConsumeMissingKeyReturnsNil(t *testing.T) {
	repo := startRedis(t)
	ctx := context.Background()

	got, err := repo.Consume(ctx, "does-not-exist")
	if err != nil {
		t.Fatalf("consume missing: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil record for missing key, got %+v", got)
	}
}
