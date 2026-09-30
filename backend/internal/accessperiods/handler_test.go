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

type handlerTestService struct {
	createFn  func(context.Context, int64, int64, string, int64, int64, CreateInput) (AccessPeriod, error)
	listFn    func(context.Context, int64, int64, string, int64, int64) ([]AccessPeriod, error)
	currentFn func(context.Context, int64, int64, string, int64, int64, time.Time) (*AccessPeriod, error)
}

func (s *handlerTestService) Create(ctx context.Context, collegeID, actorID int64, role string, groupID, subgroupID int64, input CreateInput) (AccessPeriod, error) {
	return s.createFn(ctx, collegeID, actorID, role, groupID, subgroupID, input)
}

func (s *handlerTestService) List(ctx context.Context, collegeID, actorID int64, role string, groupID, subgroupID int64) ([]AccessPeriod, error) {
	return s.listFn(ctx, collegeID, actorID, role, groupID, subgroupID)
}

func (s *handlerTestService) Current(ctx context.Context, collegeID, actorID int64, role string, groupID, subgroupID int64, at time.Time) (*AccessPeriod, error) {
	return s.currentFn(ctx, collegeID, actorID, role, groupID, subgroupID, at)
}

func handlerTokenService() *auth.TokenService {
	return auth.NewTokenService("01234567890123456789012345678901", 15*time.Minute)
}

func authenticatedRequest(t *testing.T, handler http.Handler, method, path, role, body string) *httptest.ResponseRecorder {
	t.Helper()

	tokens := handlerTokenService()
	token, err := tokens.Generate(auth.User{
		ID:        42,
		CollegeID: 7,
		Role:      role,
	})
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	auth.AuthMiddleware(tokens, handler).ServeHTTP(recorder, req)
	return recorder
}

func TestCreateAccessPeriodHandlerSuccess(t *testing.T) {
	service := &handlerTestService{
		createFn: func(_ context.Context, collegeID, actorID int64, role string, groupID, subgroupID int64, input CreateInput) (AccessPeriod, error) {
			if collegeID != 7 || actorID != 42 || role != "faculty" || groupID != 100 || subgroupID != 200 {
				t.Fatalf("unexpected service arguments: college=%d actor=%d role=%s group=%d subgroup=%d", collegeID, actorID, role, groupID, subgroupID)
			}
			if input.StartsAt.IsZero() || input.EndsAt.IsZero() || !input.EndsAt.After(input.StartsAt) {
				t.Fatal("expected parsed valid time range")
			}
			return AccessPeriod{
				ID: 1, CollegeID: 7, SubgroupID: 200,
				StartsAt: "2026-09-30T10:00:00Z",
				EndsAt: "2026-09-30T12:00:00Z",
				CreatedBy: 42,
				CreatedAt: "2026-09-30T09:00:00Z",
			}, nil
		},
	}

	recorder := authenticatedRequest(
		t, NewHandler(service), http.MethodPost,
		"/groups/100/subgroups/200/access-periods", "faculty",
		`{"starts_at":"2026-09-30T10:00:00Z","ends_at":"2026-09-30T12:00:00Z"}`,
	)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateAccessPeriodRejectsStudent(t *testing.T) {
	service := &handlerTestService{
		createFn: func(context.Context, int64, int64, string, int64, int64, CreateInput) (AccessPeriod, error) {
			t.Fatal("service should not be called")
			return AccessPeriod{}, nil
		},
	}

	recorder := authenticatedRequest(
		t, NewHandler(service), http.MethodPost,
		"/groups/100/subgroups/200/access-periods", "student",
		`{"starts_at":"2026-09-30T10:00:00Z","ends_at":"2026-09-30T12:00:00Z"}`,
	)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestCreateAccessPeriodRejectsInvalidTime(t *testing.T) {
	service := &handlerTestService{
		createFn: func(context.Context, int64, int64, string, int64, int64, CreateInput) (AccessPeriod, error) {
			t.Fatal("service should not be called")
			return AccessPeriod{}, nil
		},
	}

	recorder := authenticatedRequest(
		t, NewHandler(service), http.MethodPost,
		"/groups/100/subgroups/200/access-periods", "college_admin",
		`{"starts_at":"not-a-time","ends_at":"2026-09-30T12:00:00Z"}`,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestCreateAccessPeriodMapsOverlapConflict(t *testing.T) {
	service := &handlerTestService{
		createFn: func(context.Context, int64, int64, string, int64, int64, CreateInput) (AccessPeriod, error) {
			return AccessPeriod{}, ErrPeriodOverlap
		},
	}

	recorder := authenticatedRequest(
		t, NewHandler(service), http.MethodPost,
		"/groups/100/subgroups/200/access-periods", "college_admin",
		`{"starts_at":"2026-09-30T10:00:00Z","ends_at":"2026-09-30T12:00:00Z"}`,
	)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", recorder.Code)
	}
}

func TestListAccessPeriodsSuccess(t *testing.T) {
	service := &handlerTestService{
		listFn: func(_ context.Context, collegeID, actorID int64, role string, groupID, subgroupID int64) ([]AccessPeriod, error) {
			if collegeID != 7 || actorID != 42 || role != "student" || groupID != 100 || subgroupID != 200 {
				t.Fatalf("unexpected service arguments")
			}
			return []AccessPeriod{{ID: 1, CollegeID: 7, SubgroupID: 200}}, nil
		},
	}

	recorder := authenticatedRequest(
		t, NewHandler(service), http.MethodGet,
		"/groups/100/subgroups/200/access-periods", "student", "",
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var periods []AccessPeriod
	if err := json.Unmarshal(recorder.Body.Bytes(), &periods); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(periods) != 1 || periods[0].ID != 1 {
		t.Fatalf("unexpected periods: %+v", periods)
	}
}

func TestCurrentAccessPeriodOpen(t *testing.T) {
	service := &handlerTestService{
		currentFn: func(_ context.Context, collegeID, actorID int64, role string, groupID, subgroupID int64, at time.Time) (*AccessPeriod, error) {
			if collegeID != 7 || actorID != 42 || role != "student" || groupID != 100 || subgroupID != 200 {
				t.Fatalf("unexpected service arguments")
			}
			if at.IsZero() {
				t.Fatal("expected handler to provide current time")
			}
			return &AccessPeriod{ID: 1, CollegeID: 7, SubgroupID: 200}, nil
		},
	}

	recorder := authenticatedRequest(
		t, NewHandler(service), http.MethodGet,
		"/groups/100/subgroups/200/access-periods/current", "student", "",
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var response struct {
		IsOpen bool          `json:"is_open"`
		Period *AccessPeriod `json:"period"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.IsOpen || response.Period == nil || response.Period.ID != 1 {
		t.Fatalf("unexpected current response: %+v", response)
	}
}

func TestCurrentAccessPeriodClosed(t *testing.T) {
	service := &handlerTestService{
		currentFn: func(context.Context, int64, int64, string, int64, int64, time.Time) (*AccessPeriod, error) {
			return nil, nil
		},
	}

	recorder := authenticatedRequest(
		t, NewHandler(service), http.MethodGet,
		"/groups/100/subgroups/200/access-periods/current", "student", "",
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var response struct {
		IsOpen bool          `json:"is_open"`
		Period *AccessPeriod `json:"period"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.IsOpen || response.Period != nil {
		t.Fatalf("expected closed response, got %+v", response)
	}
}

func TestAccessPeriodHandlerUnauthorized(t *testing.T) {
	service := &handlerTestService{}
	req := httptest.NewRequest(
		http.MethodGet,
		"/groups/100/subgroups/200/access-periods/current",
		nil,
	)
	recorder := httptest.NewRecorder()

	NewHandler(service).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestAccessPeriodHandlerMapsServiceErrors(t *testing.T) {
	tests := []struct {
		name string
		err error
		want int
	}{
		{"invalid input", ErrInvalidInput, http.StatusBadRequest},
		{"forbidden", ErrForbidden, http.StatusForbidden},
		{"group missing", ErrGroupNotFound, http.StatusNotFound},
		{"subgroup missing", ErrSubgroupNotFound, http.StatusNotFound},
		{"internal", errors.New("database unavailable"), http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			service := &handlerTestService{
				listFn: func(context.Context, int64, int64, string, int64, int64) ([]AccessPeriod, error) {
					return nil, tc.err
				},
			}

			recorder := authenticatedRequest(
				t, NewHandler(service), http.MethodGet,
				"/groups/100/subgroups/200/access-periods", "student", "",
			)
			if recorder.Code != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, recorder.Code)
			}
		})
	}
}
