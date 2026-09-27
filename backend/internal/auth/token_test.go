package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testJWTSecret = "test-only-secret-at-least-32-bytes-long"

func TestTokenServiceGenerateAndValidate(t *testing.T) {
	service := NewTokenService(testJWTSecret, 15*time.Minute)

	user := User{
		ID:          42,
		CollegeID:   7,
		FullName:    "Test Student",
		Email:       "student@example.com",
		FacultyCode: "",
		Role:        "student",
	}

	tokenString, err := service.Generate(user)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	claims, err := service.Validate(tokenString)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("UserID = %d, want %d", claims.UserID, user.ID)
	}

	if claims.CollegeID != user.CollegeID {
		t.Errorf("CollegeID = %d, want %d", claims.CollegeID, user.CollegeID)
	}

	if claims.Role != user.Role {
		t.Errorf("Role = %q, want %q", claims.Role, user.Role)
	}

	if claims.Subject != "42" {
		t.Errorf("Subject = %q, want %q", claims.Subject, "42")
	}

	if claims.ExpiresAt == nil {
		t.Fatal("ExpiresAt is nil")
	}

	if claims.ExpiresAt.Time.Before(time.Now()) {
		t.Error("generated token is already expired")
	}
}

func TestTokenServiceRejectsExpiredToken(t *testing.T) {
	service := NewTokenService(testJWTSecret, 15*time.Minute)

	claims := AccessClaims{
		UserID:    42,
		CollegeID: 7,
		Role:      "student",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "42",
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	expiredToken, err := token.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	_, err = service.Validate(expiredToken)
	if !errors.Is(err, ErrTokenExpired) {
		t.Errorf("Validate() error = %v, want ErrTokenExpired", err)
	}
}

func TestTokenServiceRejectsInvalidSignature(t *testing.T) {
	issuer := NewTokenService(testJWTSecret, 15*time.Minute)
	validator := NewTokenService(
		"another-test-secret-at-least-32-bytes-long",
		15*time.Minute,
	)

	tokenString, err := issuer.Generate(User{
		ID:        42,
		CollegeID: 7,
		Role:      "student",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	_, err = validator.Validate(tokenString)
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("Validate() error = %v, want ErrInvalidToken", err)
	}
}

func TestTokenServiceRejectsMalformedToken(t *testing.T) {
	service := NewTokenService(testJWTSecret, 15*time.Minute)

	_, err := service.Validate("not-a-jwt")
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("Validate() error = %v, want ErrInvalidToken", err)
	}
}

func TestTokenServiceRejectsNonHS256Token(t *testing.T) {
	service := NewTokenService(testJWTSecret, 15*time.Minute)

	claims := AccessClaims{
		UserID:    42,
		CollegeID: 7,
		Role:      "student",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "42",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)

	tokenString, err := token.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	_, err = service.Validate(tokenString)
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("Validate() error = %v, want ErrInvalidToken", err)
	}
}
