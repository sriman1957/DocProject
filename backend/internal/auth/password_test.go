package auth

import (
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "CorrectHorseBattery123!"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == "" {
		t.Fatal("HashPassword() returned an empty hash")
	}

	if strings.Contains(hash, password) {
		t.Fatal("password hash contains the plaintext password")
	}

	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("unexpected hash format: %q", hash)
	}
}

func TestHashPasswordGeneratesUniqueHashes(t *testing.T) {
	password := "CorrectHorseBattery123!"

	first, err := HashPassword(password)
	if err != nil {
		t.Fatalf("first HashPassword() error = %v", err)
	}

	second, err := HashPassword(password)
	if err != nil {
		t.Fatalf("second HashPassword() error = %v", err)
	}

	if first == second {
		t.Fatal("identical passwords produced identical hashes")
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "CorrectHorseBattery123!"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{
			name:     "correct password",
			password: password,
			want:     true,
		},
		{
			name:     "incorrect password",
			password: "WrongPassword123!",
			want:     false,
		},
		{
			name:     "empty password",
			password: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := VerifyPassword(tt.password, hash)
			if err != nil {
				t.Fatalf("VerifyPassword() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("VerifyPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVerifyPasswordRejectsInvalidHash(t *testing.T) {
	tests := []struct {
		name string
		hash string
	}{
		{
			name: "empty hash",
			hash: "",
		},
		{
			name: "invalid format",
			hash: "invalid-hash",
		},
		{
			name: "unsupported algorithm",
			hash: "$bcrypt$invalid",
		},
		{
			name: "invalid parameters",
			hash: "$argon2id$v=1$m=invalid,t=3,p=2$salt$hash",
		},
		{
			name: "excessive memory",
			hash: "$argon2id$v=1$m=999999999,t=3,p=2$c2FsdHNhbHQ$YWJjZGVmZ2hpamtsbW5vcA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := VerifyPassword("some-password", tt.hash)
			if err == nil {
				t.Fatal("VerifyPassword() expected an error")
			}
		})
	}
}

func TestHashPasswordRejectsEmptyPassword(t *testing.T) {
	_, err := HashPassword("")
	if err == nil {
		t.Fatal("HashPassword() expected an error for an empty password")
	}
}
