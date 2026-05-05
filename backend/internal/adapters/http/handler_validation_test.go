package http

import "testing"

func p64(v int64) *int64 { return &v }

func baseLimits() RequestLimits {
	return RequestLimits{
		MaxMetaBytes:     1024,
		MaxCipherBytes:   4096,
		MaxTTLSeconds:    3600,
		MaxViews:         100,
		MaxFileBytes:     4096,
		AllowedFileMIMEs: []string{"application/pdf", "application/octet-stream"},
	}
}

func TestValidateCreateSecretRequestAcceptsValidPayload(t *testing.T) {
	limits := baseLimits()
	views := int64(1)
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       `{"v":1,"t":"text","alg":"AES-GCM-256"}`,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "text",
		Views:      &views,
	}, limits)
	if err != nil {
		t.Fatalf("expected valid payload, got error: %v", err)
	}
}

func TestValidateCreateSecretRequestRejectsLargeMeta(t *testing.T) {
	limits := baseLimits()
	limits.MaxMetaBytes = 8
	views := int64(1)
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       "0123456789",
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "text",
		Views:      &views,
	}, limits)
	if err == nil {
		t.Fatal("expected error for oversized meta")
	}
}

func TestValidateCreateSecretRequestRejectsLargeCiphertext(t *testing.T) {
	limits := baseLimits()
	limits.MaxCipherBytes = 8
	views := int64(1)
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       `{"v":1,"t":"text","alg":"AES-GCM-256"}`,
		Ciphertext: "0123456789",
		Kind:       "text",
		Views:      &views,
	}, limits)
	if err == nil {
		t.Fatal("expected error for oversized ciphertext")
	}
}

func TestValidateCreateSecretRequestRejectsInvalidKind(t *testing.T) {
	limits := baseLimits()
	views := int64(1)
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       `{"v":1,"t":"text","alg":"AES-GCM-256"}`,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "unknown",
		Views:      &views,
	}, limits)
	if err == nil {
		t.Fatal("expected error for invalid kind")
	}
}

func TestValidateCreateSecretRequestAcceptsArgon2Meta(t *testing.T) {
	limits := baseLimits()
	limits.RequirePassword = true
	views := int64(1)
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       `{"v":1,"t":"text","alg":"AES-GCM-256","kdf":"ARGON2ID","tt":3,"tm":65536,"tp":1,"s":"YWJjZGVmZw"}`,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "text",
		Views:      &views,
	}, limits)
	if err != nil {
		t.Fatalf("expected valid Argon2 payload, got error: %v", err)
	}
}

func TestValidateCreateSecretRequestRejectsNoPasswordWhenRequired(t *testing.T) {
	limits := baseLimits()
	limits.RequirePassword = true
	views := int64(1)
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       `{"v":1,"t":"text","alg":"AES-GCM-256"}`,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "text",
		Views:      &views,
	}, limits)
	if err == nil {
		t.Fatal("expected error when password-only mode is enabled")
	}
}

func TestValidateCreateSecretRequestRejectsBadMetaSchema(t *testing.T) {
	limits := baseLimits()
	views := int64(1)
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       `{"v":1,"t":"text","alg":"AES-GCM-256","kdf":"ARGON2ID","tt":3,"tm":4096,"tp":1,"s":"abc"}`,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "text",
		Views:      &views,
	}, limits)
	if err == nil {
		t.Fatal("expected error for weak Argon2 memory parameter")
	}
}

func TestValidateCreateSecretRequestRejectsKDFParamsWithoutKDF(t *testing.T) {
	limits := baseLimits()
	views := int64(1)
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       `{"v":1,"t":"text","alg":"AES-GCM-256","s":"abc","i":310000}`,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "text",
		Views:      &views,
	}, limits)
	if err == nil {
		t.Fatal("expected error when kdf params are set without kdf")
	}
}

func TestValidateCreateSecretRequestRejectsFileTooLarge(t *testing.T) {
	limits := RequestLimits{
		MaxMetaBytes:     512,
		MaxCipherBytes:   1024,
		MaxTTLSeconds:    3600,
		MaxViews:         100,
		MaxFileBytes:     1024,
		AllowedFileMIMEs: []string{"application/pdf"},
		RequirePassword:  false,
	}
	views := int64(1)
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       `{"v":1,"t":"file","alg":"AES-GCM-256","n":"file.pdf","m":"application/pdf","z":2048}`,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "file",
		Views:      &views,
	}, limits)
	if err == nil {
		t.Fatal("expected error for oversized file metadata")
	}
}

func TestValidateCreateSecretRequestRejectsFileMimeNotAllowed(t *testing.T) {
	limits := RequestLimits{
		MaxMetaBytes:     512,
		MaxCipherBytes:   1024,
		MaxTTLSeconds:    3600,
		MaxViews:         100,
		MaxFileBytes:     4096,
		AllowedFileMIMEs: []string{"application/pdf"},
		RequirePassword:  false,
	}
	views := int64(1)
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       `{"v":1,"t":"file","alg":"AES-GCM-256","n":"image.png","m":"image/png","z":100}`,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "file",
		Views:      &views,
	}, limits)
	if err == nil {
		t.Fatal("expected error for disallowed mime type")
	}
}

func TestValidateCreateSecretRequestRejectsUnsafeFileName(t *testing.T) {
	limits := RequestLimits{
		MaxMetaBytes:     512,
		MaxCipherBytes:   1024,
		MaxTTLSeconds:    3600,
		MaxViews:         100,
		MaxFileBytes:     4096,
		AllowedFileMIMEs: []string{"application/pdf"},
		RequirePassword:  false,
	}
	views := int64(1)
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       `{"v":1,"t":"file","alg":"AES-GCM-256","n":"../secret.pdf","m":"application/pdf","z":100}`,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "file",
		Views:      &views,
	}, limits)
	if err == nil {
		t.Fatal("expected error for unsafe file name")
	}
}

func TestValidateCreateSecretRequestAcceptsArgon2MemoryAboveFloorWhenRequired(t *testing.T) {
	limits := baseLimits()
	limits.RequirePassword = true
	views := int64(1)
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       `{"v":1,"t":"text","alg":"AES-GCM-256","kdf":"ARGON2ID","tt":3,"tm":131072,"tp":1,"s":"YWJjZGVmZw"}`,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "text",
		Views:      &views,
	}, limits)
	if err != nil {
		t.Fatalf("Argon2 with tm above the strict floor must be accepted, got: %v", err)
	}
}

func TestValidateCreateSecretRequestRejectsArgon2BelowStrictFloorWhenRequired(t *testing.T) {
	limits := baseLimits()
	limits.RequirePassword = true
	views := int64(1)
	meta := `{"v":1,"t":"text","alg":"AES-GCM-256","kdf":"ARGON2ID","tt":3,"tm":32768,"tp":1,"s":"YWJjZGVmZw"}`
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       meta,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "text",
		Views:      &views,
	}, limits)
	if err == nil {
		t.Fatal("expected rejection of tm=32768 in RequirePassword mode")
	}
}

func TestValidateCreateSecretRequestAcceptsArgon2BelowStrictFloorWhenNotRequired(t *testing.T) {
	limits := baseLimits()
	limits.RequirePassword = false
	views := int64(1)
	meta := `{"v":1,"t":"text","alg":"AES-GCM-256","kdf":"ARGON2ID","tt":3,"tm":32768,"tp":1,"s":"YWJjZGVmZw"}`
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       meta,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "text",
		Views:      &views,
	}, limits)
	if err != nil {
		t.Fatalf("tm=32768 must be accepted when RequirePassword is false (proves the gating is conditional), got: %v", err)
	}
}

func TestValidateCreateSecretRequestRejectsPBKDF2WhenRequired(t *testing.T) {
	limits := baseLimits()
	limits.RequirePassword = true
	views := int64(1)
	meta := `{"v":1,"t":"text","alg":"AES-GCM-256","kdf":"PBKDF2-SHA256","i":310000,"s":"YWJjZGVmZw"}`
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       meta,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "text",
		Views:      &views,
	}, limits)
	if err == nil {
		t.Fatal("expected rejection of PBKDF2 in RequirePassword mode")
	}
}

func TestValidateCreateSecretRequestAcceptsPBKDF2WhenNotRequired(t *testing.T) {
	limits := baseLimits()
	limits.RequirePassword = false
	views := int64(1)
	meta := `{"v":1,"t":"text","alg":"AES-GCM-256","kdf":"PBKDF2-SHA256","i":310000,"s":"YWJjZGVmZw"}`
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       meta,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "text",
		Views:      &views,
	}, limits)
	if err != nil {
		t.Fatalf("PBKDF2 must remain accepted when RequirePassword is false (legacy share-link compatibility), got: %v", err)
	}
}

func TestValidateCreateSecretRequestRejectsBothViewsAndTTL(t *testing.T) {
	limits := baseLimits()
	err := validateCreateSecretRequest(createSecretRequest{
		Meta:       `{"v":1,"t":"text","alg":"AES-GCM-256"}`,
		Ciphertext: `{"iv":"abc","ct":"xyz"}`,
		Kind:       "text",
		Views:      p64(1),
		TTLSeconds: p64(120),
	}, limits)
	if err == nil {
		t.Fatal("expected error for setting views and ttl_seconds together")
	}
}
