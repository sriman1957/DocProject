package users

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"docproject/backend/internal/auth"
)

type testUserService struct {
	createFn func(
		ctx context.Context,
		collegeID int64,
		input CreateInput,
	) (User, error)
}

func (s *testUserService) Create(
	ctx context.Context,
	collegeID int64,
	input CreateInput,
) (User, error) {
	return s.createFn(ctx, collegeID, input)
}

func newTestTokenService() *auth.TokenService {
	return auth.NewTokenService(
		"01234567890123456789012345678901",
		15*time.Minute,
	)
}

func makeAuthenticatedRequest(
	t *testing.T,
	handler http.Handler,
	role string,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()

	tokens := newTestTokenService()

	token, err := tokens.Generate(auth.User{
		ID:        42,
		CollegeID: 7,
		Role:      role,
	})
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/users",
		strings.NewReader(body),
	)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	auth.AuthMiddleware(tokens, handler).ServeHTTP(recorder, req)

	return recorder
}

func validUserRequest() string {
	return `{
		"full_name": "Test Student",
		"email": "student@example.com",
		"password": "a-strong-test-password",
		"role": "student"
	}`
}

func TestCreateUserUnauthorized(t *testing.T) {
	service := &testUserService{
		createFn: func(
			context.Context,
			int64,
			CreateInput,
		) (User, error) {
			t.Fatal("service should not be called")
			return User{}, nil
		},
	}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/users",
		strings.NewReader(validUserRequest()),
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			recorder.Code,
		)
	}
}

func TestCreateUserForbiddenForNonAdmin(t *testing.T) {
	for _, role := range []string{"student", "faculty"} {
		t.Run(role, func(t *testing.T) {
			service := &testUserService{
				createFn: func(
					context.Context,
					int64,
					CreateInput,
				) (User, error) {
					t.Fatal("service should not be called")
					return User{}, nil
				},
			}

			recorder := makeAuthenticatedRequest(
				t,
				NewHandler(service),
				role,
				validUserRequest(),
			)

			if recorder.Code != http.StatusForbidden {
				t.Fatalf(
					"expected status %d, got %d",
					http.StatusForbidden,
					recorder.Code,
				)
			}
		})
	}
}

func TestCreateUserSuccess(t *testing.T) {
	service := &testUserService{
		createFn: func(
			_ context.Context,
			collegeID int64,
			input CreateInput,
		) (User, error) {
			if collegeID != 7 {
				t.Errorf("expected college ID 7, got %d", collegeID)
			}

			if input.Role != "student" {
				t.Errorf("expected student role, got %q", input.Role)
			}

			if input.Password != "a-strong-test-password" {
				t.Error("password was not passed to the service")
			}

			return User{
				ID:        100,
				CollegeID: collegeID,
				FullName:  input.FullName,
				Email:     input.Email,
				Role:      input.Role,
				IsActive:  true,
			}, nil
		},
	}

	recorder := makeAuthenticatedRequest(
		t,
		NewHandler(service),
		"college_admin",
		validUserRequest(),
	)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d. Body: %s",
			http.StatusCreated,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var user User
	if err := json.Unmarshal(recorder.Body.Bytes(), &user); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if user.ID != 100 {
		t.Errorf("expected user ID 100, got %d", user.ID)
	}

	if user.CollegeID != 7 {
		t.Errorf("expected college ID 7, got %d", user.CollegeID)
	}

	if user.Email != "student@example.com" {
		t.Errorf("unexpected email: %q", user.Email)
	}

	// Ensure the JSON response doesn't expose password-related fields.
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response map: %v", err)
	}

	if _, exists := response["password"]; exists {
		t.Error("response must not contain a password")
	}

	if _, exists := response["password_hash"]; exists {
		t.Error("response must not contain a password hash")
	}
}

func TestCreateUserRejectsCollegeIDField(t *testing.T) {
	service := &testUserService{
		createFn: func(
			context.Context,
			int64,
			CreateInput,
		) (User, error) {
			t.Fatal("service should not be called")
			return User{}, nil
		},
	}

	body := `{
		"full_name": "Test Student",
		"email": "student@example.com",
		"password": "a-strong-test-password",
		"role": "student",
		"college_id": 999
	}`

	recorder := makeAuthenticatedRequest(
		t,
		NewHandler(service),
		"college_admin",
		body,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestCreateUserInvalidJSON(t *testing.T) {
	service := &testUserService{
		createFn: func(
			context.Context,
			int64,
			CreateInput,
		) (User, error) {
			t.Fatal("service should not be called")
			return User{}, nil
		},
	}

	recorder := makeAuthenticatedRequest(
		t,
		NewHandler(service),
		"college_admin",
		`{"full_name":`,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestCreateUserDuplicateEmail(t *testing.T) {
	service := &testUserService{
		createFn: func(
			context.Context,
			int64,
			CreateInput,
		) (User, error) {
			return User{}, ErrEmailAlreadyExists
		},
	}

	recorder := makeAuthenticatedRequest(
		t,
		NewHandler(service),
		"college_admin",
		validUserRequest(),
	)

	if recorder.Code != http.StatusConflict {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusConflict,
			recorder.Code,
		)
	}
}

func TestCreateUserInvalidInput(t *testing.T) {
	service := &testUserService{
		createFn: func(
			context.Context,
			int64,
			CreateInput,
		) (User, error) {
			return User{}, ErrInvalidInput
		},
	}

	recorder := makeAuthenticatedRequest(
		t,
		NewHandler(service),
		"college_admin",
		validUserRequest(),
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestCreateUserInternalError(t *testing.T) {
	service := &testUserService{
		createFn: func(
			context.Context,
			int64,
			CreateInput,
		) (User, error) {
			return User{}, errors.New("database unavailable")
		},
	}

	recorder := makeAuthenticatedRequest(
		t,
		NewHandler(service),
		"college_admin",
		validUserRequest(),
	)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			recorder.Code,
		)
	}

	if strings.Contains(recorder.Body.String(), "database unavailable") {
		t.Error("internal error details must not be exposed")
	}
}
