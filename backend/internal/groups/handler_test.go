package groups

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

type testService struct {
	createFn func(
		ctx context.Context,
		collegeID int64,
		createdBy int64,
		input CreateInput,
	) (Group, error)

	listFn func(
		ctx context.Context,
		collegeID int64,
	) ([]Group, error)
}

func (s *testService) Create(
	ctx context.Context,
	collegeID int64,
	createdBy int64,
	input CreateInput,
) (Group, error) {
	return s.createFn(ctx, collegeID, createdBy, input)
}

func (s *testService) List(
	ctx context.Context,
	collegeID int64,
) ([]Group, error) {
	return s.listFn(ctx, collegeID)
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
		"/groups",
		strings.NewReader(body),
	)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	auth.AuthMiddleware(tokens, handler).ServeHTTP(recorder, req)

	return recorder
}

func TestCreateGroupUnauthorized(t *testing.T) {
	service := &testService{
		createFn: func(
			context.Context,
			int64,
			int64,
			CreateInput,
		) (Group, error) {
			t.Fatal("service should not be called")
			return Group{}, nil
		},
	}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/groups",
		strings.NewReader(`{"name":"IT 2024-2028"}`),
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

func TestCreateGroupForbiddenForNonAdmin(t *testing.T) {
	for _, role := range []string{"student", "faculty"} {
		t.Run(role, func(t *testing.T) {
			service := &testService{
				createFn: func(
					context.Context,
					int64,
					int64,
					CreateInput,
				) (Group, error) {
					t.Fatal("service should not be called")
					return Group{}, nil
				},
			}

			handler := NewHandler(service)

			recorder := makeAuthenticatedRequest(
				t,
				handler,
				role,
				`{"name":"IT 2024-2028"}`,
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

func TestCreateGroupInvalidJSON(t *testing.T) {
	service := &testService{
		createFn: func(
			context.Context,
			int64,
			int64,
			CreateInput,
		) (Group, error) {
			t.Fatal("service should not be called")
			return Group{}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		"college_admin",
		`{"name":`,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestCreateGroupSuccess(t *testing.T) {
	service := &testService{
		createFn: func(
			_ context.Context,
			collegeID int64,
			createdBy int64,
			input CreateInput,
		) (Group, error) {
			if collegeID != 7 {
				t.Errorf("expected college ID 7, got %d", collegeID)
			}

			if createdBy != 42 {
				t.Errorf("expected creator ID 42, got %d", createdBy)
			}

			if input.Name != "IT 2024-2028" {
				t.Errorf("unexpected name: %q", input.Name)
			}

			if input.Description != "IT department batch" {
				t.Errorf(
					"unexpected description: %q",
					input.Description,
				)
			}

			return Group{
				ID:          100,
				CollegeID:   collegeID,
				Name:        input.Name,
				Description: input.Description,
				CreatedBy:   createdBy,
				IsActive:    true,
				CreatedAt:   "2026-09-27 10:00:00+00",
				UpdatedAt:   "2026-09-27 10:00:00+00",
			}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		"college_admin",
		`{
			"name": "IT 2024-2028",
			"description": "IT department batch"
		}`,
	)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d. Body: %s",
			http.StatusCreated,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var group Group
	if err := json.Unmarshal(recorder.Body.Bytes(), &group); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if group.ID != 100 {
		t.Errorf("expected group ID 100, got %d", group.ID)
	}

	if group.CollegeID != 7 {
		t.Errorf("expected college ID 7, got %d", group.CollegeID)
	}
}

func TestCreateGroupRejectsUnknownFields(t *testing.T) {
	service := &testService{
		createFn: func(
			context.Context,
			int64,
			int64,
			CreateInput,
		) (Group, error) {
			t.Fatal("service should not be called")
			return Group{}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		"college_admin",
		`{
			"name": "IT 2024-2028",
			"college_id": 999
		}`,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestCreateGroupServiceError(t *testing.T) {
	service := &testService{
		createFn: func(
			context.Context,
			int64,
			int64,
			CreateInput,
		) (Group, error) {
			return Group{}, errors.New("database unavailable")
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		"college_admin",
		`{"name":"IT 2024-2028"}`,
	)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			recorder.Code,
		)
	}
}

func makeAuthenticatedGETRequest(
	t *testing.T,
	handler http.Handler,
	role string,
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
		http.MethodGet,
		"/groups",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+token)

	recorder := httptest.NewRecorder()

	auth.AuthMiddleware(tokens, handler).ServeHTTP(recorder, req)

	return recorder
}

func TestListGroupsSuccess(t *testing.T) {
	service := &testService{
		listFn: func(
			_ context.Context,
			collegeID int64,
		) ([]Group, error) {
			if collegeID != 7 {
				t.Errorf("expected college ID 7, got %d", collegeID)
			}

			return []Group{
				{
					ID:          100,
					CollegeID:   7,
					Name:        "IT 2024-2028",
					Description: "IT department batch",
					CreatedBy:   42,
					IsActive:    true,
				},
			}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedGETRequest(
		t,
		handler,
		"college_admin",
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d. Body: %s",
			http.StatusOK,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var groups []Group
	if err := json.Unmarshal(recorder.Body.Bytes(), &groups); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}

	if groups[0].ID != 100 {
		t.Errorf("expected group ID 100, got %d", groups[0].ID)
	}
}

func TestListGroupsForbiddenForNonAdmin(t *testing.T) {
	for _, role := range []string{"student", "faculty"} {
		t.Run(role, func(t *testing.T) {
			service := &testService{
				listFn: func(
					context.Context,
					int64,
				) ([]Group, error) {
					t.Fatal("service should not be called")
					return nil, nil
				},
			}

			handler := NewHandler(service)

			recorder := makeAuthenticatedGETRequest(
				t,
				handler,
				role,
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

func TestListGroupsServiceError(t *testing.T) {
	service := &testService{
		listFn: func(
			context.Context,
			int64,
		) ([]Group, error) {
			return nil, errors.New("database unavailable")
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedGETRequest(
		t,
		handler,
		"college_admin",
	)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			recorder.Code,
		)
	}
}
