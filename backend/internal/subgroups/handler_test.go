package subgroups

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
		actorID int64,
		actorRole string,
		groupID int64,
		input CreateInput,
	) (Subgroup, error)

	listFn func(
		ctx context.Context,
		collegeID int64,
		actorID int64,
		actorRole string,
		groupID int64,
	) ([]Subgroup, error)
}

func (s *testService) Create(
	ctx context.Context,
	collegeID int64,
	actorID int64,
	actorRole string,
	groupID int64,
	input CreateInput,
) (Subgroup, error) {
	return s.createFn(ctx, collegeID, actorID, actorRole, groupID, input)
}

func (s *testService) List(
	ctx context.Context,
	collegeID int64,
	actorID int64,
	actorRole string,
	groupID int64,
) ([]Subgroup, error) {
	return s.listFn(ctx, collegeID, actorID, actorRole, groupID)
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
	method string,
	path string,
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
		method,
		path,
		strings.NewReader(body),
	)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	auth.AuthMiddleware(tokens, handler).ServeHTTP(recorder, req)

	return recorder
}

func TestCreateSubgroupSuccess(t *testing.T) {
	service := &testService{
		createFn: func(
			_ context.Context,
			collegeID int64,
			actorID int64,
			actorRole string,
			groupID int64,
			input CreateInput,
		) (Subgroup, error) {
			if collegeID != 7 {
				t.Errorf("expected college ID 7, got %d", collegeID)
			}
			if actorID != 42 {
				t.Errorf("expected actor ID 42, got %d", actorID)
			}
			if actorRole != "faculty" {
				t.Errorf("expected role faculty, got %q", actorRole)
			}
			if groupID != 100 {
				t.Errorf("expected group ID 100, got %d", groupID)
			}
			if input.Name != "Internship" {
				t.Errorf("unexpected name: %q", input.Name)
			}

			return Subgroup{
				ID:          200,
				CollegeID:   collegeID,
				GroupID:     groupID,
				Name:        input.Name,
				Description: input.Description,
				CreatedBy:   actorID,
				IsActive:    true,
			}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		http.MethodPost,
		"/groups/100/subgroups",
		"faculty",
		`{"name":"Internship","description":"Internship certificates"}`,
	)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d. Body: %s",
			http.StatusCreated,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var subgroup Subgroup
	if err := json.Unmarshal(recorder.Body.Bytes(), &subgroup); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if subgroup.ID != 200 {
		t.Errorf("expected subgroup ID 200, got %d", subgroup.ID)
	}
	if subgroup.GroupID != 100 {
		t.Errorf("expected group ID 100, got %d", subgroup.GroupID)
	}
	if subgroup.Name != "Internship" {
		t.Errorf("expected name Internship, got %q", subgroup.Name)
	}
}

func TestCreateSubgroupUnauthorized(t *testing.T) {
	service := &testService{
		createFn: func(
			context.Context,
			int64,
			int64,
			string,
			int64,
			CreateInput,
		) (Subgroup, error) {
			t.Fatal("service should not be called")
			return Subgroup{}, nil
		},
	}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/groups/100/subgroups",
		strings.NewReader(`{"name":"Internship"}`),
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

func TestCreateSubgroupForbiddenForStudent(t *testing.T) {
	service := &testService{
		createFn: func(
			context.Context,
			int64,
			int64,
			string,
			int64,
			CreateInput,
		) (Subgroup, error) {
			t.Fatal("service should not be called")
			return Subgroup{}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		http.MethodPost,
		"/groups/100/subgroups",
		"student",
		`{"name":"Internship"}`,
	)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status %d, got %d. Body: %s",
			http.StatusForbidden,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func TestCreateSubgroupInvalidGroupID(t *testing.T) {
	testCases := []struct {
		name    string
		groupID string
	}{
		{name: "non-numeric", groupID: "abc"},
		{name: "zero", groupID: "0"},
		{name: "negative", groupID: "-1"},
		{name: "overflow", groupID: "999999999999999999999999"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := &testService{
				createFn: func(
					context.Context,
					int64,
					int64,
					string,
					int64,
					CreateInput,
				) (Subgroup, error) {
					t.Fatal("service should not be called")
					return Subgroup{}, nil
				},
			}

			handler := NewHandler(service)

			recorder := makeAuthenticatedRequest(
				t,
				handler,
				http.MethodPost,
				"/groups/"+tc.groupID+"/subgroups",
				"college_admin",
				`{"name":"Internship"}`,
			)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf(
					"expected status %d, got %d. Body: %s",
					http.StatusBadRequest,
					recorder.Code,
					recorder.Body.String(),
				)
			}
		})
	}
}

func TestCreateSubgroupInvalidJSON(t *testing.T) {
	service := &testService{
		createFn: func(
			context.Context,
			int64,
			int64,
			string,
			int64,
			CreateInput,
		) (Subgroup, error) {
			t.Fatal("service should not be called")
			return Subgroup{}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		http.MethodPost,
		"/groups/100/subgroups",
		"college_admin",
		`{"name":`,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d. Body: %s",
			http.StatusBadRequest,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func TestCreateSubgroupRejectsUnknownFields(t *testing.T) {
	service := &testService{
		createFn: func(
			context.Context,
			int64,
			int64,
			string,
			int64,
			CreateInput,
		) (Subgroup, error) {
			t.Fatal("service should not be called")
			return Subgroup{}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		http.MethodPost,
		"/groups/100/subgroups",
		"college_admin",
		`{"name":"Internship","college_id":999}`,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d. Body: %s",
			http.StatusBadRequest,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func TestCreateSubgroupServiceErrors(t *testing.T) {
	testCases := []struct {
		name           string
		serviceErr     error
		expectedStatus int
	}{
		{
			name:           "invalid input",
			serviceErr:     ErrInvalidInput,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "forbidden",
			serviceErr:     ErrForbidden,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "group not found",
			serviceErr:     ErrGroupNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "subgroup already exists",
			serviceErr:     ErrSubgroupExists,
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "unexpected error",
			serviceErr:     errors.New("database unavailable"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := &testService{
				createFn: func(
					context.Context,
					int64,
					int64,
					string,
					int64,
					CreateInput,
				) (Subgroup, error) {
					return Subgroup{}, tc.serviceErr
				},
			}

			handler := NewHandler(service)

			recorder := makeAuthenticatedRequest(
				t,
				handler,
				http.MethodPost,
				"/groups/100/subgroups",
				"college_admin",
				`{"name":"Internship"}`,
			)

			if recorder.Code != tc.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d. Body: %s",
					tc.expectedStatus,
					recorder.Code,
					recorder.Body.String(),
				)
			}
		})
	}
}

func TestListSubgroupsSuccess(t *testing.T) {
	service := &testService{
		listFn: func(
			_ context.Context,
			collegeID int64,
			actorID int64,
			actorRole string,
			groupID int64,
		) ([]Subgroup, error) {
			if collegeID != 7 {
				t.Errorf("expected college ID 7, got %d", collegeID)
			}
			if actorID != 42 {
				t.Errorf("expected actor ID 42, got %d", actorID)
			}
			if actorRole != "student" {
				t.Errorf("expected role student, got %q", actorRole)
			}
			if groupID != 100 {
				t.Errorf("expected group ID 100, got %d", groupID)
			}

			return []Subgroup{
				{
					ID:        200,
					CollegeID: collegeID,
					GroupID:   groupID,
					Name:      "Internship",
					IsActive:  true,
				},
			}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		http.MethodGet,
		"/groups/100/subgroups",
		"student",
		"",
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d. Body: %s",
			http.StatusOK,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var subgroups []Subgroup
	if err := json.Unmarshal(recorder.Body.Bytes(), &subgroups); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(subgroups) != 1 {
		t.Fatalf("expected 1 subgroup, got %d", len(subgroups))
	}
	if subgroups[0].ID != 200 {
		t.Errorf("expected subgroup ID 200, got %d", subgroups[0].ID)
	}
}

func TestListSubgroupsReturnsEmptyArray(t *testing.T) {
	service := &testService{
		listFn: func(
			context.Context,
			int64,
			int64,
			string,
			int64,
		) ([]Subgroup, error) {
			return []Subgroup{}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		http.MethodGet,
		"/groups/100/subgroups",
		"college_admin",
		"",
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d. Body: %s",
			http.StatusOK,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var subgroups []Subgroup
	if err := json.Unmarshal(recorder.Body.Bytes(), &subgroups); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if subgroups == nil {
		t.Fatal("expected an empty JSON array, got null")
	}
	if len(subgroups) != 0 {
		t.Fatalf("expected 0 subgroups, got %d", len(subgroups))
	}
}

func TestListSubgroupsInvalidGroupID(t *testing.T) {
	service := &testService{
		listFn: func(
			context.Context,
			int64,
			int64,
			string,
			int64,
		) ([]Subgroup, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		http.MethodGet,
		"/groups/abc/subgroups",
		"student",
		"",
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d. Body: %s",
			http.StatusBadRequest,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func TestListSubgroupsServiceErrors(t *testing.T) {
	testCases := []struct {
		name           string
		serviceErr     error
		expectedStatus int
	}{
		{
			name:           "invalid input",
			serviceErr:     ErrInvalidInput,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "forbidden",
			serviceErr:     ErrForbidden,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "group not found",
			serviceErr:     ErrGroupNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "unexpected error",
			serviceErr:     errors.New("database unavailable"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := &testService{
				listFn: func(
					context.Context,
					int64,
					int64,
					string,
					int64,
				) ([]Subgroup, error) {
					return nil, tc.serviceErr
				},
			}

			handler := NewHandler(service)

			recorder := makeAuthenticatedRequest(
				t,
				handler,
				http.MethodGet,
				"/groups/100/subgroups",
				"student",
				"",
			)

			if recorder.Code != tc.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d. Body: %s",
					tc.expectedStatus,
					recorder.Code,
					recorder.Body.String(),
				)
			}
		})
	}
}
