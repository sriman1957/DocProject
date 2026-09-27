package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func newTestAuthHandler(t *testing.T, row pgx.Row) http.Handler {
	t.Helper()

	service := NewService(&fakeQuerier{row: row})

	tokenService := NewTokenService(
		testJWTSecret,
		15*time.Minute,
	)

	logger := slog.New(slog.NewTextHandler(
		&bytes.Buffer{},
		nil,
	))

	return NewHandler(service, tokenService, logger)
}

func performLoginRequest(
	handler http.Handler,
	body string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(
		http.MethodPost,
		"/auth/login",
		bytes.NewBufferString(body),
	)
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	return recorder
}

func TestLoginHandlerSuccess(t *testing.T) {
	password := "StrongPassword123!"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	handler := newTestAuthHandler(t, &fakeRow{
		id:           1,
		collegeID:    10,
		fullName:     "Test Student",
		email:        "student@example.com",
		role:         "student",
		passwordHash: hash,
		isActive:     true,
	})

	body := `{
		"college_code": "COLLEGE001",
		"email": "student@example.com",
		"password": "StrongPassword123!"
	}`

	recorder := performLoginRequest(handler, body)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s",
			recorder.Code, http.StatusOK, recorder.Body.String())
	}

	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}

	var response loginResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.AccessToken == "" {
		t.Fatal("access_token is empty")
	}

	if response.User.ID != 1 {
		t.Errorf("user.ID = %d, want 1", response.User.ID)
	}

	if response.User.Email != "student@example.com" {
		t.Errorf("user.Email = %q, want student@example.com", response.User.Email)
	}

	// Verify the returned access token.
	tokenService := NewTokenService(testJWTSecret, 15*time.Minute)

	claims, err := tokenService.Validate(response.AccessToken)
	if err != nil {
		t.Fatalf("validate access token: %v", err)
	}

	if claims.UserID != response.User.ID {
		t.Errorf("token user ID = %d, want %d", claims.UserID, response.User.ID)
	}

	if bytes.Contains(recorder.Body.Bytes(), []byte(hash)) {
		t.Error("response must not expose password hash")
	}
}

func TestLoginHandlerInvalidCredentials(t *testing.T) {
	handler := newTestAuthHandler(t, &fakeRow{
		err: pgx.ErrNoRows,
	})

	body := `{
		"college_code": "COLLEGE001",
		"email": "missing@example.com",
		"password": "SomePassword123!"
	}`

	recorder := performLoginRequest(handler, body)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d",
			recorder.Code, http.StatusUnauthorized)
	}
}

func TestLoginHandlerMalformedJSON(t *testing.T) {
	handler := newTestAuthHandler(t, &fakeRow{})

	recorder := performLoginRequest(handler, `{"college_code":`)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d",
			recorder.Code, http.StatusBadRequest)
	}
}

func TestLoginHandlerRejectsUnknownFields(t *testing.T) {
	handler := newTestAuthHandler(t, &fakeRow{})

	body := `{
		"college_code": "COLLEGE001",
		"email": "student@example.com",
		"password": "Password123!",
		"unexpected": true
	}`

	recorder := performLoginRequest(handler, body)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d",
			recorder.Code, http.StatusBadRequest)
	}
}

func TestLoginHandlerRejectsMultipleJSONValues(t *testing.T) {
	handler := newTestAuthHandler(t, &fakeRow{})

	body := `{
		"college_code": "COLLEGE001",
		"email": "student@example.com",
		"password": "Password123!"
	} {}`

	recorder := performLoginRequest(handler, body)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d",
			recorder.Code, http.StatusBadRequest)
	}
}

func TestLoginHandlerRejectsEmptyInput(t *testing.T) {
	handler := newTestAuthHandler(t, &fakeRow{})

	body := `{
		"college_code": "",
		"email": "student@example.com",
		"password": "Password123!"
	}`

	recorder := performLoginRequest(handler, body)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d",
			recorder.Code, http.StatusUnauthorized)
	}
}

func TestLoginHandlerHidesInternalErrors(t *testing.T) {
	handler := newTestAuthHandler(t, &fakeRow{
		err: errors.New("database connection details"),
	})

	body := `{
		"college_code": "COLLEGE001",
		"email": "student@example.com",
		"password": "Password123!"
	}`

	recorder := performLoginRequest(handler, body)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d",
			recorder.Code, http.StatusInternalServerError)
	}

	if bytes.Contains(
		recorder.Body.Bytes(),
		[]byte("database connection details"),
	) {
		t.Error("response must not expose internal error details")
	}
}

var _ = context.Background
