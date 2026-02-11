package http

import "testing"

func TestValidateCreateSecretRequestAcceptsValidPayload(t *testing.T) {
	limits := RequestLimits{MaxMetaBytes: 64, MaxCipherBytes: 128}
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       `{"v":1,"t":"text"}`,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "text",
		TTLSeconds: 60,
	}, limits)
	if err != nil {
		t.Fatalf("expected valid payload, got error: %v", err)
	}
}

func TestValidateCreateSecretRequestRejectsLargeMeta(t *testing.T) {
	limits := RequestLimits{MaxMetaBytes: 8, MaxCipherBytes: 128}
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       "0123456789",
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "text",
		TTLSeconds: 60,
	}, limits)
	if err == nil {
		t.Fatal("expected error for oversized meta")
	}
}

func TestValidateCreateSecretRequestRejectsLargeCiphertext(t *testing.T) {
	limits := RequestLimits{MaxMetaBytes: 64, MaxCipherBytes: 8}
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       `{"v":1,"t":"text"}`,
		Ciphertext: "0123456789",
		Kind:       "text",
		TTLSeconds: 60,
	}, limits)
	if err == nil {
		t.Fatal("expected error for oversized ciphertext")
	}
}

func TestValidateCreateSecretRequestRejectsInvalidKind(t *testing.T) {
	limits := RequestLimits{MaxMetaBytes: 64, MaxCipherBytes: 128}
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       `{"v":1,"t":"text"}`,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "unknown",
		TTLSeconds: 60,
	}, limits)
	if err == nil {
		t.Fatal("expected error for invalid kind")
	}
}

