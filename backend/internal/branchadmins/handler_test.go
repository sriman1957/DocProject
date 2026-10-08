package branchadmins

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"docproject/backend/internal/auth"
)

type fakeService struct {
	assign func(
		ctx context.Context,
		collegeID int64,
		branchID int64,
		userID int64,
		assignedBy int64,
	) (Assignment, error)
}

func (f *fakeService) Assign(
	ctx context.Context,
	collegeID int64,
	branchID int64,
	userID int64,
	assignedBy int64,
) (Assignment, error) {
	return f.assign(
		ctx,
		collegeID,
		branchID,
		userID,
		assignedBy,
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
	method string,
	path string,
	body string,
	userID int64,
	collegeID int64,
	role string,
) *httptest.ResponseRecorder {
	t.Helper()

	tokens := newTestTokenService()

	token, err := tokens.Generate(auth.User{
		ID:        userID,
		CollegeID: collegeID,
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

	req.Header.Set(
		"Authorization",
		"Bearer "+token,
	)

	recorder := httptest.NewRecorder()

	auth.AuthMiddleware(
		tokens,
		handler,
	).ServeHTTP(recorder, req)

	return recorder
}

func TestHandlerRequiresAuthentication(t *testing.T) {
	service := &fakeService{
		assign: func(
			context.Context,
			int64,
			int64,
			int64,
			int64,
		) (Assignment, error) {
			t.Fatal("service should not be called")
			return Assignment{}, nil
		},
	}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/branch-admins",
		strings.NewReader(
			`{"branch_id":1,"user_id":2}`,
		),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status 401, got %d",
			recorder.Code,
		)
	}
}

func TestHandlerRequiresCollegeAdmin(t *testing.T) {
	service := &fakeService{
		assign: func(
			context.Context,
			int64,
			int64,
			int64,
			int64,
		) (Assignment, error) {
			t.Fatal("service should not be called")
			return Assignment{}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		http.MethodPost,
		"/branch-admins",
		`{"branch_id":1,"user_id":2}`,
		10,
		7,
		"faculty",
	)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status 403, got %d",
			recorder.Code,
		)
	}
}

func TestHandlerInvalidRequestBody(t *testing.T) {
	service := &fakeService{
		assign: func(
			context.Context,
			int64,
			int64,
			int64,
			int64,
		) (Assignment, error) {
			t.Fatal("service should not be called")
			return Assignment{}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		http.MethodPost,
		"/branch-admins",
		`{"branch_id":`,
		10,
		7,
		"college_admin",
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status 400, got %d",
			recorder.Code,
		)
	}
}

func TestHandlerSuccessfulAssignment(t *testing.T) {
	var (
		gotCollegeID  int64
		gotBranchID   int64
		gotUserID     int64
		gotAssignedBy int64
	)

	service := &fakeService{
		assign: func(
			_ context.Context,
			collegeID int64,
			branchID int64,
			userID int64,
			assignedBy int64,
		) (Assignment, error) {
			gotCollegeID = collegeID
			gotBranchID = branchID
			gotUserID = userID
			gotAssignedBy = assignedBy

			return Assignment{
				ID:         100,
				CollegeID:  7,
				BranchID:   1,
				UserID:     20,
				AssignedBy: 10,
				IsActive:   true,
				CreatedAt:  "2026-10-08 09:00:00+00",
				UpdatedAt:  "2026-10-08 09:00:00+00",
			}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		http.MethodPost,
		"/branch-admins",
		`{"branch_id":1,"user_id":20}`,
		10,
		7,
		"college_admin",
	)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected status 201, got %d",
			recorder.Code,
		)
	}

	if gotCollegeID != 7 {
		t.Errorf(
			"expected college ID 7, got %d",
			gotCollegeID,
		)
	}

	if gotBranchID != 1 {
		t.Errorf(
			"expected branch ID 1, got %d",
			gotBranchID,
		)
	}

	if gotUserID != 20 {
		t.Errorf(
			"expected user ID 20, got %d",
			gotUserID,
		)
	}

	if gotAssignedBy != 10 {
		t.Errorf(
			"expected assigned_by 10, got %d",
			gotAssignedBy,
		)
	}
}

func TestHandlerMapsErrors(t *testing.T) {
	tests := []struct {
		name       string
		serviceErr error
		wantStatus int
	}{
		{
			name:       "invalid input",
			serviceErr: ErrInvalidInput,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "branch not found",
			serviceErr: ErrBranchNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "user not found",
			serviceErr: ErrUserNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "duplicate assignment",
			serviceErr: ErrAssignmentExists,
			wantStatus: http.StatusConflict,
		},
		{
			name:       "unexpected error",
			serviceErr: errors.New("database failure"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeService{
				assign: func(
					context.Context,
					int64,
					int64,
					int64,
					int64,
				) (Assignment, error) {
					return Assignment{}, test.serviceErr
				},
			}

			handler := NewHandler(service)

			recorder := makeAuthenticatedRequest(
				t,
				handler,
				http.MethodPost,
				"/branch-admins",
				`{"branch_id":1,"user_id":20}`,
				10,
				7,
				"college_admin",
			)

			if recorder.Code != test.wantStatus {
				t.Fatalf(
					"expected status %d, got %d",
					test.wantStatus,
					recorder.Code,
				)
			}
		})
	}
}
