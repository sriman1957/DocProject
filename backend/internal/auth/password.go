package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonMemory      uint32 = 64 * 1024
	argonIterations  uint32 = 3
	argonParallelism uint8  = 2
	saltLength              = 16
	keyLength               = 32
)

const hashVersion = 19

// HashPassword generates an Argon2id password hash.
// Store the returned string in users.password_hash.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password must not be empty")
	}

	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		argonIterations,
		argonMemory,
		argonParallelism,
		keyLength,
	)

	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		hashVersion,
		argonMemory,
		argonIterations,
		argonParallelism,
		encodedSalt,
		encodedHash,
	)

	return encoded, nil
}

// VerifyPassword checks a password against a stored Argon2id hash.
func VerifyPassword(password, encodedHash string) (bool, error) {
	if password == "" {
		return false, nil
	}

	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 ||
		parts[1] != "argon2id" {
		return false, fmt.Errorf("invalid password hash format")
	}

	versionPart := strings.TrimPrefix(parts[2], "v=")
	version, err := strconv.Atoi(versionPart)
	if err != nil || version != hashVersion {
		return false, fmt.Errorf("unsupported Argon2 version")
	}

	var memory, iterations uint32
	var parallelism uint8

	params := strings.Split(parts[3], ",")
	if len(params) != 3 {
		return false, fmt.Errorf("invalid Argon2 parameters")
	}

	for _, param := range params {
		keyValue := strings.SplitN(param, "=", 2)
		if len(keyValue) != 2 {
			return false, fmt.Errorf("invalid Argon2 parameter")
		}

		switch keyValue[0] {
		case "m":
			value, err := strconv.ParseUint(keyValue[1], 10, 32)
			if err != nil {
				return false, fmt.Errorf("invalid Argon2 memory")
			}
			memory = uint32(value)

		case "t":
			value, err := strconv.ParseUint(keyValue[1], 10, 32)
			if err != nil {
				return false, fmt.Errorf("invalid Argon2 iterations")
			}
			iterations = uint32(value)

		case "p":
			value, err := strconv.ParseUint(keyValue[1], 10, 8)
			if err != nil {
				return false, fmt.Errorf("invalid Argon2 parallelism")
			}
			parallelism = uint8(value)

		default:
			return false, fmt.Errorf("unknown Argon2 parameter")
		}
	}

	// Reject unreasonable parameters before allocating memory.
	if memory < 8*uint32(parallelism) ||
		memory > 256*1024 ||
		iterations == 0 ||
		iterations > 10 ||
		parallelism == 0 ||
		parallelism > 8 {
		return false, fmt.Errorf("Argon2 parameters out of range")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 8 || len(salt) > 64 {
		return false, fmt.Errorf("invalid password salt")
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expectedHash) < 16 || len(expectedHash) > 64 {
		return false, fmt.Errorf("invalid password hash")
	}

	actualHash := argon2.IDKey(
		[]byte(password),
		salt,
		iterations,
		memory,
		parallelism,
		uint32(len(expectedHash)),
	)

	return subtle.ConstantTimeCompare(actualHash, expectedHash) == 1, nil
}
