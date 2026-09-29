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

	addMemberFn func(
		ctx context.Context,
		collegeID int64,
		groupID int64,
		input AddMemberInput,
	) (Member, error)

	listMembersFn func(
		ctx context.Context,
		collegeID int64,
		groupID int64,
	) ([]Member, error)
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

func (s *testService) AddMember(
	ctx context.Context,
	collegeID int64,
	groupID int64,
	input AddMemberInput,
) (Member, error) {
	if s.addMemberFn == nil {
		return Member{}, nil
	}

	return s.addMemberFn(ctx, collegeID, groupID, input)
}

func (s *testService) ListMembers(
	ctx context.Context,
	collegeID int64,
	groupID int64,
) ([]Member, error) {
	if s.listMembersFn == nil {
		return nil, nil
	}

	return s.listMembersFn(ctx, collegeID, groupID)
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

func makeAuthenticatedListMembersRequest(
	t *testing.T,
	handler http.Handler,
	role string,
	groupID string,
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
		"/groups/"+groupID+"/members",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+token)

	recorder := httptest.NewRecorder()

	auth.AuthMiddleware(tokens, handler).ServeHTTP(recorder, req)

	return recorder
}

func makeAuthenticatedMemberRequest(
	t *testing.T,
	handler http.Handler,
	role string,
	groupID string,
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
		"/groups/"+groupID+"/members",
		strings.NewReader(body),
	)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

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

func TestAddGroupMemberSuccess(t *testing.T) {
	service := &testService{
		addMemberFn: func(
			_ context.Context,
			collegeID int64,
			groupID int64,
			input AddMemberInput,
		) (Member, error) {
			if collegeID != 7 {
				t.Errorf("expected college ID 7, got %d", collegeID)
			}

			if groupID != 100 {
				t.Errorf("expected group ID 100, got %d", groupID)
			}

			if input.UserID != 55 {
				t.Errorf("expected user ID 55, got %d", input.UserID)
			}

			return Member{
				UserID:         55,
				FullName:       "Test Student",
				Email:          "student@test.example",
				Role:           "student",
				MembershipRole: "student",
				JoinedAt:       "2026-09-29 10:00:00+00",
			}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedMemberRequest(
		t,
		handler,
		"college_admin",
		"100",
		`{"user_id":55}`,
	)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d. Body: %s",
			http.StatusCreated,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var member Member
	if err := json.Unmarshal(recorder.Body.Bytes(), &member); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if member.UserID != 55 {
		t.Errorf("expected user ID 55, got %d", member.UserID)
	}

	if member.FullName != "Test Student" {
		t.Errorf("expected full name %q, got %q", "Test Student", member.FullName)
	}

	if member.MembershipRole != "student" {
		t.Errorf(
			"expected membership role %q, got %q",
			"student",
			member.MembershipRole,
		)
	}
}

func TestAddGroupMemberForbiddenForNonAdmin(t *testing.T) {
	for _, role := range []string{"student", "faculty"} {
		t.Run(role, func(t *testing.T) {
			service := &testService{
				addMemberFn: func(
					context.Context,
					int64,
					int64,
					AddMemberInput,
				) (Member, error) {
					t.Fatal("service should not be called")
					return Member{}, nil
				},
			}

			handler := NewHandler(service)

			recorder := makeAuthenticatedMemberRequest(
				t,
				handler,
				role,
				"100",
				`{"user_id":55}`,
			)

			if recorder.Code != http.StatusForbidden {
				t.Fatalf(
					"expected status %d, got %d. Body: %s",
					http.StatusForbidden,
					recorder.Code,
					recorder.Body.String(),
				)
			}
		})
	}
}

func TestAddGroupMemberInvalidGroupID(t *testing.T) {
	testCases := []struct {
		name    string
		groupID string
	}{
		{
			name:    "non-numeric group ID",
			groupID: "abc",
		},
		{
			name:    "zero group ID",
			groupID: "0",
		},
		{
			name:    "negative group ID",
			groupID: "-1",
		},
		{
			name:    "group ID overflow",
			groupID: "999999999999999999999999",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := &testService{
				addMemberFn: func(
					context.Context,
					int64,
					int64,
					AddMemberInput,
				) (Member, error) {
					t.Fatal("service should not be called")
					return Member{}, nil
				},
			}

			handler := NewHandler(service)

			recorder := makeAuthenticatedMemberRequest(
				t,
				handler,
				"college_admin",
				tc.groupID,
				`{"user_id":55}`,
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

func TestAddGroupMemberServiceErrors(t *testing.T) {
	testCases := []struct {
		name           string
		serviceErr     error
		expectedStatus int
	}{
		{
			name:           "group not found",
			serviceErr:     ErrGroupNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "eligible user not found",
			serviceErr:     ErrMemberNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "membership already exists",
			serviceErr:     ErrMembershipExists,
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "unexpected service error",
			serviceErr:     errors.New("database unavailable"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := &testService{
				addMemberFn: func(
					context.Context,
					int64,
					int64,
					AddMemberInput,
				) (Member, error) {
					return Member{}, tc.serviceErr
				},
			}

			handler := NewHandler(service)

			recorder := makeAuthenticatedMemberRequest(
				t,
				handler,
				"college_admin",
				"100",
				`{"user_id":55}`,
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

func TestListGroupMembersEmpty(t *testing.T) {
	service := &testService{
		listMembersFn: func(
			_ context.Context,
			collegeID int64,
			groupID int64,
		) ([]Member, error) {
			if collegeID != 7 {
				t.Errorf("expected college ID 7, got %d", collegeID)
			}

			if groupID != 100 {
				t.Errorf("expected group ID 100, got %d", groupID)
			}

			return []Member{}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedListMembersRequest(
		t,
		handler,
		"college_admin",
		"100",
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d. Body: %s",
			http.StatusOK,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var members []Member
	if err := json.Unmarshal(recorder.Body.Bytes(), &members); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if members == nil {
		t.Fatal("expected an empty JSON array, got null")
	}

	if len(members) != 0 {
		t.Fatalf("expected 0 members, got %d", len(members))
	}
}

func TestListGroupMembersUnauthorized(t *testing.T) {
	service := &testService{
		listMembersFn: func(
			context.Context,
			int64,
			int64,
		) ([]Member, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/groups/100/members",
		nil,
	)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d. Body: %s",
			http.StatusUnauthorized,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func TestListGroupMembersForbiddenForNonAdmin(t *testing.T) {
	for _, role := range []string{"student", "faculty"} {
		t.Run(role, func(t *testing.T) {
			service := &testService{
				listMembersFn: func(
					context.Context,
					int64,
					int64,
				) ([]Member, error) {
					t.Fatal("service should not be called")
					return nil, nil
				},
			}

			handler := NewHandler(service)

			recorder := makeAuthenticatedListMembersRequest(
				t,
				handler,
				role,
				"100",
			)

			if recorder.Code != http.StatusForbidden {
				t.Fatalf(
					"expected status %d, got %d. Body: %s",
					http.StatusForbidden,
					recorder.Code,
					recorder.Body.String(),
				)
			}
		})
	}
}

func TestListGroupMembersInvalidGroupID(t *testing.T) {
	testCases := []struct {
		name    string
		groupID string
	}{
		{
			name:    "non-numeric ID",
			groupID: "abc",
		},
		{
			name:    "zero ID",
			groupID: "0",
		},
		{
			name:    "negative ID",
			groupID: "-1",
		},
		{
			name:    "overflow ID",
			groupID: "999999999999999999999999",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := &testService{
				listMembersFn: func(
					context.Context,
					int64,
					int64,
				) ([]Member, error) {
					t.Fatal("service should not be called")
					return nil, nil
				},
			}

			handler := NewHandler(service)

			recorder := makeAuthenticatedListMembersRequest(
				t,
				handler,
				"college_admin",
				tc.groupID,
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

func TestListGroupMembersServiceErrors(t *testing.T) {
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
			name:           "group not found",
			serviceErr:     ErrGroupNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "unexpected service error",
			serviceErr:     errors.New("database unavailable"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := &testService{
				listMembersFn: func(
					context.Context,
					int64,
					int64,
				) ([]Member, error) {
					return nil, tc.serviceErr
				},
			}

			handler := NewHandler(service)

			recorder := makeAuthenticatedListMembersRequest(
				t,
				handler,
				"college_admin",
				"100",
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
