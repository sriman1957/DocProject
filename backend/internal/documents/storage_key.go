package documents

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"strings"

	"github.com/google/uuid"
)

func CalculateSHA256(data []byte) string {
	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}

func GenerateStorageKey() string {
	return fmt.Sprintf(
		"documents/%s",
		uuid.New().String(),
	)
}

func IsSafeStorageKey(key string) bool {
	if key == "" {
		return false
	}

	if strings.ContainsRune(key, '\\') {
		return false
	}

	cleaned := path.Clean(key)

	if cleaned != key {
		return false
	}

	if strings.HasPrefix(cleaned, "/") {
		return false
	}

	if cleaned == "." || cleaned == ".." {
		return false
	}

	parts := strings.Split(cleaned, "/")

	for _, part := range parts {
		if part == ".." {
			return false
		}
	}

	return strings.HasPrefix(cleaned, "documents/")
}
