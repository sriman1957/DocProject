package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuthMiddlewareRejectsMissingToken(t *testing.T) {
	tokens := NewTokenService(testJWTSecret, 15*time.Minute)

	handler := AuthMiddleware(tokens, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	))

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d",
			recorder.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddlewareRejectsInvalidToken(t *testing.T) {
	tokens := NewTokenService(testJWTSecret, 15*time.Minute)

	handler := AuthMiddleware(tokens, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	))

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer invalid.token.value")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d",
			recorder.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddlewareRejectsExpiredToken(t *testing.T) {
	tokens := NewTokenService(testJWTSecret, 15*time.Minute)

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

	tokenString, err := token.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	handler := AuthMiddleware(tokens, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	))

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+tokenString)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d",
			recorder.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddlewarePassesAuthenticatedClaims(t *testing.T) {
	tokens := NewTokenService(testJWTSecret, 15*time.Minute)

	tokenString, err := tokens.Generate(User{
		ID:        42,
		CollegeID: 7,
		Role:      "student",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	handler := AuthMiddleware(tokens, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				t.Error("expected claims in request context")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if claims.UserID != 42 {
				t.Errorf("UserID = %d, want 42", claims.UserID)
			}

			if claims.CollegeID != 7 {
				t.Errorf("CollegeID = %d, want 7", claims.CollegeID)
			}

			if claims.Role != "student" {
				t.Errorf("Role = %q, want student", claims.Role)
			}

			w.WriteHeader(http.StatusOK)
		},
	))

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+tokenString)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, want %d",
			recorder.Code, http.StatusOK)
	}
}

func TestAuthMiddlewareRejectsInvalidClaims(t *testing.T) {
	tokens := NewTokenService(testJWTSecret, 15*time.Minute)

	tokenString, err := tokens.Generate(User{
		ID:        0,
		CollegeID: 7,
		Role:      "student",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	handler := AuthMiddleware(tokens, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	))

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+tokenString)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d",
			recorder.Code, http.StatusUnauthorized)
	}
}
