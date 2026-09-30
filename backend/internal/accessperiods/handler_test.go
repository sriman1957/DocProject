package accessperiods

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
		subgroupID int64,
		input CreateInput,
	) (AccessPeriod, error)

	listFn func(
		ctx context.Context,
		collegeID int64,
		actorID int64,
		actorRole string,
		groupID int64,
		subgroupID int64,
	) ([]AccessPeriod, error)
}

func (s *testService) Create(
	ctx context.Context,
	collegeID int64,
	actorID int64,
	actorRole string,
	groupID int64,
	subgroupID int64,
	input CreateInput,
) (AccessPeriod, error) {
	return s.createFn(
		ctx,
		collegeID,
		actorID,
		actorRole,
		groupID,
		subgroupID,
		input,
	)
}

func (s *testService) List(
	ctx context.Context,
	collegeID int64,
	actorID int64,
	actorRole string,
	groupID int64,
	subgroupID int64,
) ([]AccessPeriod, error) {
	return s.listFn(
		ctx,
		collegeID,
		actorID,
		actorRole,
		groupID,
		subgroupID,
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

	req.Header.Set(
		"Authorization",
		"Bearer "+token,
	)

	req.Header.Set(
		"Content-Type",
		"application/json",
	)

	recorder := httptest.NewRecorder()

	auth.AuthMiddleware(
		tokens,
		handler,
	).ServeHTTP(recorder, req)

	return recorder
}

func TestCreateAccessPeriodSuccess(t *testing.T) {
	t.Parallel()

	service := &testService{
		createFn: func(
			_ context.Context,
			collegeID int64,
			actorID int64,
			actorRole string,
			groupID int64,
			subgroupID int64,
			input CreateInput,
		) (AccessPeriod, error) {
			if collegeID != 7 {
				t.Errorf(
					"expected college ID 7, got %d",
					collegeID,
				)
			}

			if actorID != 42 {
				t.Errorf(
					"expected actor ID 42, got %d",
					actorID,
				)
			}

			if actorRole != "college_admin" {
				t.Errorf(
					"expected role college_admin, got %s",
					actorRole,
				)
			}

			if groupID != 100 {
				t.Errorf(
					"expected group ID 100, got %d",
					groupID,
				)
			}

			if subgroupID != 200 {
				t.Errorf(
					"expected subgroup ID 200, got %d",
					subgroupID,
				)
			}

			if input.StartsAt != "2026-10-01T10:00:00Z" {
				t.Errorf(
					"unexpected starts_at: %q",
					input.StartsAt,
				)
			}

			if input.EndsAt != "2026-10-01T12:00:00Z" {
				t.Errorf(
					"unexpected ends_at: %q",
					input.EndsAt,
				)
			}

			return AccessPeriod{
				ID:         300,
				SubgroupID: subgroupID,
				CollegeID:  collegeID,
				StartsAt:   input.StartsAt,
				EndsAt:     input.EndsAt,
				CreatedBy:  actorID,
				CreatedAt:  "2026-10-01 09:00:00+00",
			}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		http.MethodPost,
		"/groups/100/subgroups/200/access-periods",
		"college_admin",
		`{
			"starts_at": "2026-10-01T10:00:00Z",
			"ends_at": "2026-10-01T12:00:00Z"
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

	var period AccessPeriod

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&period,
	); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if period.ID != 300 {
		t.Errorf(
			"expected period ID 300, got %d",
			period.ID,
		)
	}

	if period.SubgroupID != 200 {
		t.Errorf(
			"expected subgroup ID 200, got %d",
			period.SubgroupID,
		)
	}

	if period.CollegeID != 7 {
		t.Errorf(
			"expected college ID 7, got %d",
			period.CollegeID,
		)
	}

	if period.StartsAt != "2026-10-01T10:00:00Z" {
		t.Errorf(
			"unexpected starts_at: %q",
			period.StartsAt,
		)
	}

	if period.EndsAt != "2026-10-01T12:00:00Z" {
		t.Errorf(
			"unexpected ends_at: %q",
			period.EndsAt,
		)
	}
}

func TestCreateAccessPeriodFacultyRoleForwarded(t *testing.T) {
	t.Parallel()

	service := &testService{
		createFn: func(
			_ context.Context,
			_ int64,
			_ int64,
			actorRole string,
			_ int64,
			_ int64,
			input CreateInput,
		) (AccessPeriod, error) {
			if actorRole != "faculty" {
				t.Errorf(
					"expected faculty role, got %s",
					actorRole,
				)
			}

			return AccessPeriod{
				ID:        300,
				CollegeID: 7,
				StartsAt:  input.StartsAt,
				EndsAt:    input.EndsAt,
				CreatedBy: 42,
			}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		http.MethodPost,
		"/groups/100/subgroups/200/access-periods",
		"faculty",
		`{
			"starts_at": "2026-10-01T10:00:00Z",
			"ends_at": "2026-10-01T12:00:00Z"
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
}

func TestCreateAccessPeriodUnauthorized(t *testing.T) {
	t.Parallel()

	service := &testService{
		createFn: func(
			context.Context,
			int64,
			int64,
			string,
			int64,
			int64,
			CreateInput,
		) (AccessPeriod, error) {
			t.Fatal("service should not be called")
			return AccessPeriod{}, nil
		},
	}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/groups/100/subgroups/200/access-periods",
		strings.NewReader(`{
			"starts_at": "2026-10-01T10:00:00Z",
			"ends_at": "2026-10-01T12:00:00Z"
		}`),
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

func TestCreateAccessPeriodInvalidGroupID(t *testing.T) {
	t.Parallel()

	service := &testService{
		createFn: func(
			context.Context,
			int64,
			int64,
			string,
			int64,
			int64,
			CreateInput,
		) (AccessPeriod, error) {
			t.Fatal("service should not be called")
			return AccessPeriod{}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		http.MethodPost,
		"/groups/abc/subgroups/200/access-periods",
		"college_admin",
		`{
			"starts_at": "2026-10-01T10:00:00Z",
			"ends_at": "2026-10-01T12:00:00Z"
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

func TestCreateAccessPeriodInvalidSubgroupID(t *testing.T) {
	t.Parallel()

	service := &testService{
		createFn: func(
			context.Context,
			int64,
			int64,
			string,
			int64,
			int64,
			CreateInput,
		) (AccessPeriod, error) {
			t.Fatal("service should not be called")
			return AccessPeriod{}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		http.MethodPost,
		"/groups/100/subgroups/abc/access-periods",
		"college_admin",
		`{
			"starts_at": "2026-10-01T10:00:00Z",
			"ends_at": "2026-10-01T12:00:00Z"
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

func TestCreateAccessPeriodInvalidJSON(t *testing.T) {
	t.Parallel()

	service := &testService{
		createFn: func(
			context.Context,
			int64,
			int64,
			string,
			int64,
			int64,
			CreateInput,
		) (AccessPeriod, error) {
			t.Fatal("service should not be called")
			return AccessPeriod{}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		http.MethodPost,
		"/groups/100/subgroups/200/access-periods",
		"college_admin",
		`{"starts_at":`,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestCreateAccessPeriodRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	service := &testService{
		createFn: func(
			context.Context,
			int64,
			int64,
			string,
			int64,
			int64,
			CreateInput,
		) (AccessPeriod, error) {
			t.Fatal("service should not be called")
			return AccessPeriod{}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		http.MethodPost,
		"/groups/100/subgroups/200/access-periods",
		"college_admin",
		`{
			"starts_at": "2026-10-01T10:00:00Z",
			"ends_at": "2026-10-01T12:00:00Z",
			"unexpected": true
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

func TestCreateAccessPeriodServiceErrors(t *testing.T) {
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
			name:           "subgroup not found",
			serviceErr:     ErrSubgroupNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "overlap",
			serviceErr:     ErrAccessPeriodOverlap,
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
					int64,
					CreateInput,
				) (AccessPeriod, error) {
					return AccessPeriod{}, tc.serviceErr
				},
			}

			handler := NewHandler(service)

			recorder := makeAuthenticatedRequest(
				t,
				handler,
				http.MethodPost,
				"/groups/100/subgroups/200/access-periods",
				"college_admin",
				`{
					"starts_at": "2026-10-01T10:00:00Z",
					"ends_at": "2026-10-01T12:00:00Z"
				}`,
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

func TestListAccessPeriodsSuccess(t *testing.T) {
	t.Parallel()

	service := &testService{
		listFn: func(
			_ context.Context,
			collegeID int64,
			actorID int64,
			actorRole string,
			groupID int64,
			subgroupID int64,
		) ([]AccessPeriod, error) {
			if collegeID != 7 {
				t.Errorf(
					"expected college ID 7, got %d",
					collegeID,
				)
			}

			if actorID != 42 {
				t.Errorf(
					"expected actor ID 42, got %d",
					actorID,
				)
			}

			if actorRole != "college_admin" {
				t.Errorf(
					"expected role college_admin, got %s",
					actorRole,
				)
			}

			if groupID != 100 {
				t.Errorf(
					"expected group ID 100, got %d",
					groupID,
				)
			}

			if subgroupID != 200 {
				t.Errorf(
					"expected subgroup ID 200, got %d",
					subgroupID,
				)
			}

			return []AccessPeriod{
				{
					ID:         300,
					SubgroupID: 200,
					CollegeID:  7,
					StartsAt:   "2026-10-01 10:00:00+00",
					EndsAt:     "2026-10-01 12:00:00+00",
					CreatedBy:  42,
				},
				{
					ID:         301,
					SubgroupID: 200,
					CollegeID:  7,
					StartsAt:   "2026-10-01 12:00:00+00",
					EndsAt:     "2026-10-01 14:00:00+00",
					CreatedBy:  42,
				},
			}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		http.MethodGet,
		"/groups/100/subgroups/200/access-periods",
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

	var periods []AccessPeriod

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&periods,
	); err != nil {
		t.Fatalf(
			"decode response: %v",
			err,
		)
	}

	if len(periods) != 2 {
		t.Fatalf(
			"expected 2 access periods, got %d",
			len(periods),
		)
	}

	if periods[0].ID != 300 {
		t.Errorf(
			"expected first period ID 300, got %d",
			periods[0].ID,
		)
	}

	if periods[1].ID != 301 {
		t.Errorf(
			"expected second period ID 301, got %d",
			periods[1].ID,
		)
	}
}

func TestListAccessPeriodsReturnsEmptyArray(t *testing.T) {
	t.Parallel()

	service := &testService{
		listFn: func(
			context.Context,
			int64,
			int64,
			string,
			int64,
			int64,
		) ([]AccessPeriod, error) {
			return []AccessPeriod{}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		http.MethodGet,
		"/groups/100/subgroups/200/access-periods",
		"college_admin",
		"",
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	var periods []AccessPeriod

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&periods,
	); err != nil {
		t.Fatalf(
			"decode response: %v",
			err,
		)
	}

	if periods == nil {
		t.Fatal("expected empty JSON array, got null")
	}

	if len(periods) != 0 {
		t.Fatalf(
			"expected 0 access periods, got %d",
			len(periods),
		)
	}
}

func TestListAccessPeriodsServiceErrors(t *testing.T) {
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
			name:           "subgroup not found",
			serviceErr:     ErrSubgroupNotFound,
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
					int64,
				) ([]AccessPeriod, error) {
					return nil, tc.serviceErr
				},
			}

			handler := NewHandler(service)

			recorder := makeAuthenticatedRequest(
				t,
				handler,
				http.MethodGet,
				"/groups/100/subgroups/200/access-periods",
				"college_admin",
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

func TestListAccessPeriodsInvalidGroupID(t *testing.T) {
	t.Parallel()

	service := &testService{
		listFn: func(
			context.Context,
			int64,
			int64,
			string,
			int64,
			int64,
		) ([]AccessPeriod, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		http.MethodGet,
		"/groups/abc/subgroups/200/access-periods",
		"college_admin",
		"",
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestListAccessPeriodsInvalidSubgroupID(t *testing.T) {
	t.Parallel()

	service := &testService{
		listFn: func(
			context.Context,
			int64,
			int64,
			string,
			int64,
			int64,
		) ([]AccessPeriod, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeAuthenticatedRequest(
		t,
		handler,
		http.MethodGet,
		"/groups/100/subgroups/abc/access-periods",
		"college_admin",
		"",
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestListAccessPeriodsUnauthorized(t *testing.T) {
	t.Parallel()

	service := &testService{
		listFn: func(
			context.Context,
			int64,
			int64,
			string,
			int64,
			int64,
		) ([]AccessPeriod, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/groups/100/subgroups/200/access-periods",
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
