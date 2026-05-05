package redis

import (
	"strings"
	"testing"
)

func TestDecodeConsumeResultReturnsNilForNil(t *testing.T) {
	record, err := decodeConsumeResult(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record != nil {
		t.Fatalf("expected nil record, got %+v", record)
	}
}

func TestDecodeConsumeResultRejectsNonString(t *testing.T) {
	cases := []struct {
		name  string
		input any
	}{
		{"int64", int64(42)},
		{"float64", 1.5},
		{"slice", []any{"a", "b"}},
		{"bool", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			record, err := decodeConsumeResult(c.input)
			if record != nil {
				t.Fatalf("non-string input must yield nil record, got %+v", record)
			}
			if err == nil || !strings.Contains(err.Error(), "unexpected payload type") {
				t.Fatalf("expected 'unexpected payload type' error, got %v", err)
			}
		})
	}
}

func TestDecodeConsumeResultParsesValidPayload(t *testing.T) {
	payload := `{"meta":"meta-blob","ciphertext":"cipher-blob","kind":"text"}`
	record, err := decodeConsumeResult(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record == nil {
		t.Fatal("expected record, got nil")
	}
	if record.Meta != "meta-blob" || record.Ciphertext != "cipher-blob" || record.Kind != "text" {
		t.Fatalf("unexpected record: %+v", record)
	}
}

func TestDecodeConsumeResultRejectsMalformedJSON(t *testing.T) {
	record, err := decodeConsumeResult("{not json")
	if record != nil {
		t.Fatalf("malformed JSON must yield nil record, got %+v", record)
	}
	if err == nil {
		t.Fatal("expected JSON error, got nil")
	}
}
