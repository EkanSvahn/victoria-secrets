package domain

import "errors"

type SecretKind string

const (
	SecretKindText SecretKind = "text"
	SecretKindFile SecretKind = "file"
)

type Secret struct {
	Meta       string
	Ciphertext string
	Kind       SecretKind
}

type CreateSecretInput struct {
	Meta       string
	Ciphertext string
	Kind       SecretKind
	TTLSeconds *int64
	Views      *int64
}

func (in CreateSecretInput) Validate(maxTTLSeconds int64, maxViews int64) error {
	if in.Meta == "" {
		return errors.New("meta is required")
	}
	if in.Ciphertext == "" {
		return errors.New("ciphertext is required")
	}
	if in.Kind != SecretKindText && in.Kind != SecretKindFile {
		return errors.New("kind must be text or file")
	}
	if in.TTLSeconds != nil && in.Views != nil {
		return errors.New("views and ttl_seconds cannot both be set")
	}
	if in.TTLSeconds != nil {
		if *in.TTLSeconds < 60 || *in.TTLSeconds > maxTTLSeconds {
			return errors.New("ttl_seconds out of allowed range")
		}
	}
	if in.Views != nil {
		if *in.Views < 1 || *in.Views > maxViews {
			return errors.New("views out of allowed range")
		}
	}
	return nil
}
