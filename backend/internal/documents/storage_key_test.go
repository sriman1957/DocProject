package documents

import (
	"strings"
	"testing"
)

func TestCalculateSHA256(t *testing.T) {
	t.Parallel()

	data := []byte("hello world")

	hash := CalculateSHA256(data)

	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"

	if hash != expected {
		t.Fatalf(
			"expected SHA-256 %s, got %s",
			expected,
			hash,
		)
	}
}

func TestCalculateSHA256EmptyData(t *testing.T) {
	t.Parallel()

	hash := CalculateSHA256(nil)

	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	if hash != expected {
		t.Fatalf(
			"expected SHA-256 %s, got %s",
			expected,
			hash,
		)
	}
}

func TestCalculateSHA256DifferentDataProducesDifferentHashes(t *testing.T) {
	t.Parallel()

	first := CalculateSHA256([]byte("certificate one"))
	second := CalculateSHA256([]byte("certificate two"))

	if first == second {
		t.Fatal("expected different data to produce different hashes")
	}
}

func TestCalculateSHA256ReturnsLowercaseHex(t *testing.T) {
	t.Parallel()

	hash := CalculateSHA256([]byte("test"))

	if len(hash) != 64 {
		t.Fatalf(
			"expected 64-character hash, got %d",
			len(hash),
		)
	}

	if hash != strings.ToLower(hash) {
		t.Fatalf(
			"expected lowercase hash, got %s",
			hash,
		)
	}

	for _, char := range hash {
		if !strings.ContainsRune(
			"0123456789abcdef",
			char,
		) {
			t.Fatalf(
				"hash contains non-hex character %q",
				char,
			)
		}
	}
}

func TestGenerateStorageKey(t *testing.T) {
	t.Parallel()

	key := GenerateStorageKey()

	if !strings.HasPrefix(key, "documents/") {
		t.Fatalf(
			"expected documents/ prefix, got %s",
			key,
		)
	}

	if !IsSafeStorageKey(key) {
		t.Fatalf(
			"generated storage key is not safe: %s",
			key,
		)
	}
}

func TestGenerateStorageKeyProducesUniqueKeys(t *testing.T) {
	t.Parallel()

	first := GenerateStorageKey()
	second := GenerateStorageKey()

	if first == second {
		t.Fatalf(
			"expected unique storage keys, got %s",
			first,
		)
	}
}

func TestIsSafeStorageKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		safe bool
	}{
		{
			name: "valid key",
			key:  "documents/550e8400-e29b-41d4-a716-446655440000",
			safe: true,
		},
		{
			name: "empty key",
			key:  "",
			safe: false,
		},
		{
			name: "absolute unix path",
			key:  "/documents/file",
			safe: false,
		},
		{
			name: "parent traversal",
			key:  "documents/../secret",
			safe: false,
		},
		{
			name: "nested traversal",
			key:  "documents/user/../../secret",
			safe: false,
		},
		{
			name: "windows traversal",
			key:  `documents\..\secret`,
			safe: false,
		},
		{
			name: "wrong directory",
			key:  "uploads/file",
			safe: false,
		},
		{
			name: "current directory",
			key:  ".",
			safe: false,
		},
		{
			name: "parent directory",
			key:  "..",
			safe: false,
		},
		{
			name: "double dot filename is allowed",
			key:  "documents/..hidden/file",
			safe: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSafeStorageKey(tt.key); got != tt.safe {
				t.Fatalf(
					"IsSafeStorageKey(%q) = %v, want %v",
					tt.key,
					got,
					tt.safe,
				)
			}
		})
	}
}
