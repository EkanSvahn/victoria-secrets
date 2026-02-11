package http

import "testing"

func TestValidateCreateSecretRequestAcceptsValidPayload(t *testing.T) {
	limits := RequestLimits{MaxMetaBytes: 64, MaxCipherBytes: 128}
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       `{"v":1,"t":"text","alg":"AES-GCM-256"}`,
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
		Meta:       `{"v":1,"t":"text","alg":"AES-GCM-256"}`,
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
		Meta:       `{"v":1,"t":"text","alg":"AES-GCM-256"}`,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "unknown",
		TTLSeconds: 60,
	}, limits)
	if err == nil {
		t.Fatal("expected error for invalid kind")
	}
}

func TestValidateCreateSecretRequestAcceptsArgon2Meta(t *testing.T) {
	limits := RequestLimits{MaxMetaBytes: 256, MaxCipherBytes: 1024, RequirePassword: true}
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       `{"v":1,"t":"text","alg":"AES-GCM-256","kdf":"ARGON2ID","tt":3,"tm":65536,"tp":1,"s":"YWJjZGVmZw"}`,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "text",
		TTLSeconds: 60,
	}, limits)
	if err != nil {
		t.Fatalf("expected valid Argon2 payload, got error: %v", err)
	}
}

func TestValidateCreateSecretRequestRejectsNoPasswordWhenRequired(t *testing.T) {
	limits := RequestLimits{MaxMetaBytes: 128, MaxCipherBytes: 128, RequirePassword: true}
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       `{"v":1,"t":"text","alg":"AES-GCM-256"}`,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "text",
		TTLSeconds: 60,
	}, limits)
	if err == nil {
		t.Fatal("expected error when password-only mode is enabled")
	}
}

func TestValidateCreateSecretRequestRejectsBadMetaSchema(t *testing.T) {
	limits := RequestLimits{MaxMetaBytes: 256, MaxCipherBytes: 128}
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       `{"v":1,"t":"text","alg":"AES-GCM-256","kdf":"ARGON2ID","tt":3,"tm":4096,"tp":1,"s":"abc"}`,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "text",
		TTLSeconds: 60,
	}, limits)
	if err == nil {
		t.Fatal("expected error for weak Argon2 memory parameter")
	}
}

func TestValidateCreateSecretRequestRejectsKDFParamsWithoutKDF(t *testing.T) {
	limits := RequestLimits{MaxMetaBytes: 256, MaxCipherBytes: 128}
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       `{"v":1,"t":"text","alg":"AES-GCM-256","s":"abc","i":310000}`,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "text",
		TTLSeconds: 60,
	}, limits)
	if err == nil {
		t.Fatal("expected error when kdf params are set without kdf")
	}
}

func TestValidateCreateSecretRequestRejectsFileTooLarge(t *testing.T) {
	limits := RequestLimits{
		MaxMetaBytes:      512,
		MaxCipherBytes:    1024,
		MaxFileBytes:      1024,
		AllowedFileMIMEs:  []string{"application/pdf"},
		RequirePassword:   false,
	}
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       `{"v":1,"t":"file","alg":"AES-GCM-256","n":"file.pdf","m":"application/pdf","z":2048}`,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "file",
		TTLSeconds: 60,
	}, limits)
	if err == nil {
		t.Fatal("expected error for oversized file metadata")
	}
}

func TestValidateCreateSecretRequestRejectsFileMimeNotAllowed(t *testing.T) {
	limits := RequestLimits{
		MaxMetaBytes:      512,
		MaxCipherBytes:    1024,
		MaxFileBytes:      4096,
		AllowedFileMIMEs:  []string{"application/pdf"},
		RequirePassword:   false,
	}
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       `{"v":1,"t":"file","alg":"AES-GCM-256","n":"image.png","m":"image/png","z":100}`,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "file",
		TTLSeconds: 60,
	}, limits)
	if err == nil {
		t.Fatal("expected error for disallowed mime type")
	}
}

func TestValidateCreateSecretRequestRejectsUnsafeFileName(t *testing.T) {
	limits := RequestLimits{
		MaxMetaBytes:      512,
		MaxCipherBytes:    1024,
		MaxFileBytes:      4096,
		AllowedFileMIMEs:  []string{"application/pdf"},
		RequirePassword:   false,
	}
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       `{"v":1,"t":"file","alg":"AES-GCM-256","n":"../secret.pdf","m":"application/pdf","z":100}`,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "file",
		TTLSeconds: 60,
	}, limits)
	if err == nil {
		t.Fatal("expected error for unsafe file name")
	}
}
