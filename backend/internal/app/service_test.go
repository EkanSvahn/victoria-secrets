package app

import (
	"context"
	"errors"
	"testing"

	"victora-secret-code/backend/internal/domain"
	"victora-secret-code/backend/internal/ports"
)

type fakeRepo struct {
	storeData map[string]ports.SecretRecord
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{storeData: map[string]ports.SecretRecord{}}
}

func (f *fakeRepo) Store(_ context.Context, id string, payload ports.SecretRecord, _ int64) error {
	if _, exists := f.storeData[id]; exists {
		return errors.New("exists")
	}
	f.storeData[id] = payload
	return nil
}

func (f *fakeRepo) Consume(_ context.Context, id string) (*ports.SecretRecord, error) {
	v, ok := f.storeData[id]
	if !ok {
		return nil, nil
	}
	delete(f.storeData, id)
	return &v, nil
}

func (f *fakeRepo) Exists(_ context.Context, id string) (bool, error) {
	_, ok := f.storeData[id]
	return ok, nil
}

func TestCreateAndConsumeSecret(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, 3600, 24)
	id, err := svc.CreateSecret(context.Background(), domain.CreateSecretInput{
		Meta:       "{}",
		Ciphertext: "abc",
		Kind:       domain.SecretKindText,
		TTLSeconds: 600,
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if id == "" {
		t.Fatal("expected id")
	}
	secret, err := svc.ConsumeSecret(context.Background(), id)
	if err != nil {
		t.Fatalf("consume failed: %v", err)
	}
	if secret.Ciphertext != "abc" {
		t.Fatalf("unexpected payload: %s", secret.Ciphertext)
	}
	_, err = svc.ConsumeSecret(context.Background(), id)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestValidation(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, 3600, 24)
	_, err := svc.CreateSecret(context.Background(), domain.CreateSecretInput{
		Meta:       "",
		Ciphertext: "abc",
		Kind:       domain.SecretKindText,
		TTLSeconds: 600,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}
