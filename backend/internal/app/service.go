package app

import (
	"context"
	"errors"

	"victora-secret-code/backend/internal/domain"
	"victora-secret-code/backend/internal/ports"
	"victora-secret-code/backend/internal/security"
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("secret not found")
)

type Service struct {
	repo         ports.SecretRepository
	maxTTL       int64
	maxViews     int64
	idBytes      int
	maxIDRetries int
}

func NewService(repo ports.SecretRepository, maxTTL int64, maxViews int64, idBytes int) *Service {
	return &Service{repo: repo, maxTTL: maxTTL, maxViews: maxViews, idBytes: idBytes, maxIDRetries: 5}
}

func (s *Service) CreateSecret(ctx context.Context, in domain.CreateSecretInput) (string, error) {
	if in.TTLSeconds == nil && in.Views == nil {
		defaultViews := int64(1)
		in.Views = &defaultViews
	}
	if err := in.Validate(s.maxTTL, s.maxViews); err != nil {
		return "", ErrInvalidInput
	}

	record := ports.SecretRecord{
		Meta:           in.Meta,
		Ciphertext:     in.Ciphertext,
		Kind:           string(in.Kind),
		ViewsRemaining: in.Views,
	}
	for i := 0; i < s.maxIDRetries; i++ {
		id, err := security.NewURLSafeID(s.idBytes)
		if err != nil {
			return "", err
		}
		if err := s.repo.Store(ctx, id, record, in.TTLSeconds); err == nil {
			return id, nil
		}
	}
	return "", errors.New("failed to allocate unique secret id")
}

func (s *Service) ConsumeSecret(ctx context.Context, id string) (*domain.Secret, error) {
	if id == "" {
		return nil, ErrInvalidInput
	}
	record, err := s.repo.Consume(ctx, id)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, ErrNotFound
	}
	kind := domain.SecretKind(record.Kind)
	if kind != domain.SecretKindText && kind != domain.SecretKindFile {
		return nil, ErrInvalidInput
	}
	return &domain.Secret{Meta: record.Meta, Ciphertext: record.Ciphertext, Kind: kind}, nil
}

func (s *Service) PreviewSecret(ctx context.Context, id string) (bool, error) {
	if id == "" {
		return false, ErrInvalidInput
	}
	return s.repo.Exists(ctx, id)
}
