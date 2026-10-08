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
		role string,
		input CreateInput,
	) (Group, error)

	listFn func(
		ctx context.Context,
		collegeID int64,
		userID int64,
		role string,
	) ([]Group, error)

	addMemberFn func(
		ctx context.Context,
		collegeID int64,
		userID int64,
		role string,
		groupID int64,
		input AddMemberInput,
	) (Member, error)

	listMembersFn func(
		ctx context.Context,
		collegeID int64,
		userID int64,
		role string,
		groupID int64,
	) ([]Member, error)
}

func (s *testService) Create(
	ctx context.Context,
	collegeID int64,
	createdBy int64,
	role string,
	input CreateInput,
) (Group, error) {
	if s.createFn == nil {
		return Group{}, nil
	}

	return s.createFn(
		ctx,
		collegeID,
		createdBy,
		role,
		input,
	)
}

func (s *testService) List(
	ctx context.Context,
	collegeID int64,
	userID int64,
	role string,
) ([]Group, error) {
	if s.listFn == nil {
		return nil, nil
	}

	return s.listFn(
		ctx,
		collegeID,
		userID,
		role,
	)
}

func (s *testService) AddMember(
	ctx context.Context,
	collegeID int64,
	userID int64,
	role string,
	groupID int64,
	input AddMemberInput,
) (Member, error) {
	if s.addMemberFn == nil {
		return Member{}, nil
	}

	return s.addMemberFn(
		ctx,
		collegeID,
		userID,
		role,
		groupID,
		input,
	)
}

func (s *testService) ListMembers(
	ctx context.Context,
	collegeID int64,
	userID int64,
	role string,
	groupID int64,
) ([]Member, error) {
	if s.listMembersFn == nil {
		return nil, nil
	}

	return s.listMembersFn(
		ctx,
		collegeID,
		userID,
		role,
		groupID,
	)
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

	auth.AuthMiddleware(tokens, handler).ServeHTTP(
		recorder,
		req,
	)

	return recorder
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

	auth.AuthMiddleware(tokens, handler).ServeHTTP(
		recorder,
		req,
	)

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

	auth.AuthMiddleware(tokens, handler).ServeHTTP(
		recorder,
		req,
	)

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

	auth.AuthMiddleware(tokens, handler).ServeHTTP(
		recorder,
		req,
	)

	return recorder
}

// -----------------------------------------------------------------------------
// Create group
// -----------------------------------------------------------------------------

func TestCreateGroupUnauthorized(t *testing.T) {
	service := &testService{
		createFn: func(
			context.Context,
			int64,
			int64,
			string,
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
		strings.NewReader(`{"branch_id":1,"name":"IT 2024-2028"}`),
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

func TestCreateGroupForbiddenForStudent(t *testing.T) {
	service := &testService{
		createFn: func(
			context.Context,
			int64,
			int64,
			string,
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
		"student",
		`{"branch_id":1,"name":"IT 2024-2028"}`,
	)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusForbidden,
			recorder.Code,
		)
	}
}

func TestCreateGroupAllowsFacultyToReachService(t *testing.T) {
	called := false

	service := &testService{
		createFn: func(
			_ context.Context,
			collegeID int64,
			createdBy int64,
			role string,
			input CreateInput,
		) (Group, error) {
			called = true

			if collegeID != 7 {
				t.Errorf("expected college ID 7, got %d", collegeID)
			}

			if createdBy != 42 {
				t.Errorf("expected creator ID 42, got %d", createdBy)
			}

			if role != "faculty" {
				t.Errorf("expected role faculty, got %q", role)
			}

			if input.BranchID != 3 {
				t.Errorf("expected branch ID 3, got %d", input.BranchID)
			}

			return Group{
				ID:        100,
				CollegeID: 7,
				BranchID:  3,
				Name:      "IT 2024-2028",
				CreatedBy: 42,
				IsActive:  true,
			}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		"faculty",
		`{
			"branch_id": 3,
			"name": "IT 2024-2028"
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

	if !called {
		t.Fatal("expected service to be called for faculty")
	}
}

func TestCreateGroupInvalidJSON(t *testing.T) {
	service := &testService{
		createFn: func(
			context.Context,
			int64,
			int64,
			string,
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

func TestCreateGroupRejectsUnknownFields(t *testing.T) {
	service := &testService{
		createFn: func(
			context.Context,
			int64,
			int64,
			string,
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
			"branch_id": 3,
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

func TestCreateGroupRejectsMultipleJSONValues(t *testing.T) {
	service := &testService{
		createFn: func(
			context.Context,
			int64,
			int64,
			string,
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
		`{"branch_id":3,"name":"IT 2024-2028"} {"branch_id":4}`,
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
			role string,
			input CreateInput,
		) (Group, error) {
			if collegeID != 7 {
				t.Errorf("expected college ID 7, got %d", collegeID)
			}

			if createdBy != 42 {
				t.Errorf("expected creator ID 42, got %d", createdBy)
			}

			if role != "college_admin" {
				t.Errorf("expected role college_admin, got %q", role)
			}

			if input.BranchID != 3 {
				t.Errorf("expected branch ID 3, got %d", input.BranchID)
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
				BranchID:    input.BranchID,
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
			"branch_id": 3,
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

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&group,
	); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if group.ID != 100 {
		t.Errorf("expected group ID 100, got %d", group.ID)
	}

	if group.CollegeID != 7 {
		t.Errorf(
			"expected group college ID 7, got %d",
			group.CollegeID,
		)
	}

	if group.BranchID != 3 {
		t.Errorf(
			"expected group branch ID 3, got %d",
			group.BranchID,
		)
	}
}

func TestCreateGroupBranchNotFound(t *testing.T) {
	service := &testService{
		createFn: func(
			context.Context,
			int64,
			int64,
			string,
			CreateInput,
		) (Group, error) {
			return Group{}, ErrBranchNotFound
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		"college_admin",
		`{"branch_id":999,"name":"IT 2024-2028"}`,
	)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d. Body: %s",
			http.StatusNotFound,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func TestCreateGroupForbiddenFromService(t *testing.T) {
	service := &testService{
		createFn: func(
			context.Context,
			int64,
			int64,
			string,
			CreateInput,
		) (Group, error) {
			return Group{}, ErrForbidden
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		"faculty",
		`{"branch_id":3,"name":"IT 2024-2028"}`,
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

func TestCreateGroupInvalidInput(t *testing.T) {
	service := &testService{
		createFn: func(
			context.Context,
			int64,
			int64,
			string,
			CreateInput,
		) (Group, error) {
			return Group{}, ErrInvalidInput
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		"college_admin",
		`{"branch_id":0,"name":""}`,
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
			string,
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
		`{"branch_id":3,"name":"IT 2024-2028"}`,
	)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			recorder.Code,
		)
	}
}

// -----------------------------------------------------------------------------
// List groups
// -----------------------------------------------------------------------------

func TestListGroupsSuccess(t *testing.T) {
	service := &testService{
		listFn: func(
			_ context.Context,
			collegeID int64,
			userID int64,
			role string,
		) ([]Group, error) {
			if collegeID != 7 {
				t.Errorf("expected college ID 7, got %d", collegeID)
			}

			if userID != 42 {
				t.Errorf("expected user ID 42, got %d", userID)
			}

			if role != "college_admin" {
				t.Errorf("expected role college_admin, got %q", role)
			}

			return []Group{
				{
					ID:          100,
					CollegeID:   7,
					BranchID:    3,
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

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&groups,
	); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}

	if groups[0].ID != 100 {
		t.Errorf(
			"expected group ID 100, got %d",
			groups[0].ID,
		)
	}

	if groups[0].BranchID != 3 {
		t.Errorf(
			"expected branch ID 3, got %d",
			groups[0].BranchID,
		)
	}
}

func TestListGroupsAllowsFacultyToReachService(t *testing.T) {
	called := false

	service := &testService{
		listFn: func(
			_ context.Context,
			collegeID int64,
			userID int64,
			role string,
		) ([]Group, error) {
			called = true

			if collegeID != 7 {
				t.Errorf("expected college ID 7, got %d", collegeID)
			}

			if userID != 42 {
				t.Errorf("expected user ID 42, got %d", userID)
			}

			if role != "faculty" {
				t.Errorf("expected role faculty, got %q", role)
			}

			return []Group{}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedGETRequest(
		t,
		handler,
		"faculty",
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	if !called {
		t.Fatal("expected service to be called for faculty")
	}
}

func TestListGroupsAllowsStudentToReachService(t *testing.T) {
	called := false

	service := &testService{
		listFn: func(
			_ context.Context,
			collegeID int64,
			userID int64,
			role string,
		) ([]Group, error) {
			called = true

			if collegeID != 7 {
				t.Errorf("expected college ID 7, got %d", collegeID)
			}

			if userID != 42 {
				t.Errorf("expected user ID 42, got %d", userID)
			}

			if role != "student" {
				t.Errorf("expected role student, got %q", role)
			}

			return []Group{
				{
					ID:          100,
					CollegeID:   7,
					BranchID:    3,
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
		"student",
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d. Body: %s",
			http.StatusOK,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if !called {
		t.Fatal("expected service to be called for student")
	}

	var groups []Group

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&groups,
	); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}

	if groups[0].ID != 100 {
		t.Errorf(
			"expected group ID 100, got %d",
			groups[0].ID,
		)
	}
}

func TestListGroupsForbiddenFromService(t *testing.T) {
	service := &testService{
		listFn: func(
			context.Context,
			int64,
			int64,
			string,
		) ([]Group, error) {
			return nil, ErrForbidden
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedGETRequest(
		t,
		handler,
		"faculty",
	)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusForbidden,
			recorder.Code,
		)
	}
}

func TestListGroupsInvalidInput(t *testing.T) {
	service := &testService{
		listFn: func(
			context.Context,
			int64,
			int64,
			string,
		) ([]Group, error) {
			return nil, ErrInvalidInput
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedGETRequest(
		t,
		handler,
		"college_admin",
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestListGroupsServiceError(t *testing.T) {
	service := &testService{
		listFn: func(
			context.Context,
			int64,
			int64,
			string,
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

// -----------------------------------------------------------------------------
// Add group member
// -----------------------------------------------------------------------------

func TestAddGroupMemberSuccess(t *testing.T) {
	service := &testService{
		addMemberFn: func(
			_ context.Context,
			collegeID int64,
			userID int64,
			role string,
			groupID int64,
			input AddMemberInput,
		) (Member, error) {
			if collegeID != 7 {
				t.Errorf("expected college ID 7, got %d", collegeID)
			}

			if userID != 42 {
				t.Errorf("expected caller user ID 42, got %d", userID)
			}

			if role != "college_admin" {
				t.Errorf("expected role college_admin, got %q", role)
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

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&member,
	); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if member.UserID != 55 {
		t.Errorf(
			"expected user ID 55, got %d",
			member.UserID,
		)
	}

	if member.FullName != "Test Student" {
		t.Errorf(
			"expected full name %q, got %q",
			"Test Student",
			member.FullName,
		)
	}

	if member.MembershipRole != "student" {
		t.Errorf(
			"expected membership role %q, got %q",
			"student",
			member.MembershipRole,
		)
	}
}

func TestAddGroupMemberAllowsFacultyToReachService(t *testing.T) {
	called := false

	service := &testService{
		addMemberFn: func(
			_ context.Context,
			collegeID int64,
			userID int64,
			role string,
			groupID int64,
			input AddMemberInput,
		) (Member, error) {
			called = true

			if collegeID != 7 {
				t.Errorf("expected college ID 7, got %d", collegeID)
			}

			if userID != 42 {
				t.Errorf("expected caller user ID 42, got %d", userID)
			}

			if role != "faculty" {
				t.Errorf("expected role faculty, got %q", role)
			}

			if groupID != 100 {
				t.Errorf("expected group ID 100, got %d", groupID)
			}

			if input.UserID != 55 {
				t.Errorf("expected target user ID 55, got %d", input.UserID)
			}

			return Member{
				UserID:         55,
				FullName:       "Test Student",
				Email:          "student@test.example",
				Role:           "student",
				MembershipRole: "student",
			}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedMemberRequest(
		t,
		handler,
		"faculty",
		"100",
		`{"user_id":55}`,
	)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusCreated,
			recorder.Code,
		)
	}

	if !called {
		t.Fatal("expected service to be called for faculty")
	}
}

func TestAddGroupMemberForbiddenForStudent(t *testing.T) {
	service := &testService{
		addMemberFn: func(
			context.Context,
			int64,
			int64,
			string,
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
		"student",
		"100",
		`{"user_id":55}`,
	)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusForbidden,
			recorder.Code,
		)
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
					string,
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

func TestAddGroupMemberInvalidJSON(t *testing.T) {
	service := &testService{
		addMemberFn: func(
			context.Context,
			int64,
			int64,
			string,
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
		"100",
		`{"user_id":`,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestAddGroupMemberRejectsUnknownFields(t *testing.T) {
	service := &testService{
		addMemberFn: func(
			context.Context,
			int64,
			int64,
			string,
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
		"100",
		`{"user_id":55,"college_id":999}`,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestAddGroupMemberRejectsMultipleJSONValues(t *testing.T) {
	service := &testService{
		addMemberFn: func(
			context.Context,
			int64,
			int64,
			string,
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
		"100",
		`{"user_id":55} {"user_id":56}`,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestAddGroupMemberServiceErrors(t *testing.T) {
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
					string,
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

// -----------------------------------------------------------------------------
// List group members
// -----------------------------------------------------------------------------

func TestListGroupMembersEmpty(t *testing.T) {
	service := &testService{
		listMembersFn: func(
			_ context.Context,
			collegeID int64,
			userID int64,
			role string,
			groupID int64,
		) ([]Member, error) {
			if collegeID != 7 {
				t.Errorf("expected college ID 7, got %d", collegeID)
			}

			if userID != 42 {
				t.Errorf("expected user ID 42, got %d", userID)
			}

			if role != "college_admin" {
				t.Errorf("expected role college_admin, got %q", role)
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

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&members,
	); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if members == nil {
		t.Fatal("expected an empty JSON array, got null")
	}

	if len(members) != 0 {
		t.Fatalf(
			"expected 0 members, got %d",
			len(members),
		)
	}
}

func TestListGroupMembersAllowsFacultyToReachService(t *testing.T) {
	called := false

	service := &testService{
		listMembersFn: func(
			_ context.Context,
			collegeID int64,
			userID int64,
			role string,
			groupID int64,
		) ([]Member, error) {
			called = true

			if collegeID != 7 {
				t.Errorf("expected college ID 7, got %d", collegeID)
			}

			if userID != 42 {
				t.Errorf("expected user ID 42, got %d", userID)
			}

			if role != "faculty" {
				t.Errorf("expected role faculty, got %q", role)
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
		"faculty",
		"100",
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	if !called {
		t.Fatal("expected service to be called for faculty")
	}
}

func TestListGroupMembersUnauthorized(t *testing.T) {
	service := &testService{
		listMembersFn: func(
			context.Context,
			int64,
			int64,
			string,
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
			"expected status %d, got %d",
			http.StatusUnauthorized,
			recorder.Code,
		)
	}
}

func TestListGroupMembersForbiddenForStudent(t *testing.T) {
	service := &testService{
		listMembersFn: func(
			context.Context,
			int64,
			int64,
			string,
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
		"student",
		"100",
	)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusForbidden,
			recorder.Code,
		)
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
					string,
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
			name:           "forbidden",
			serviceErr:     ErrForbidden,
			expectedStatus: http.StatusForbidden,
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
					string,
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
