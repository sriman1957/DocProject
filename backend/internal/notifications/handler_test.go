package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"docproject/backend/internal/auth"
)

type mockNotificationHandlerService struct {
	notifications      []Notification
	err                error
	gotCollegeID       int64
	gotUserID          int64
	calls              int
	markedNotification Notification
	markReadErr        error
	gotNotificationID  int64
	unreadCount        int64
	unreadCountErr     error
}

func (m *mockNotificationHandlerService) List(
	_ context.Context,
	collegeID int64,
	userID int64,
) ([]Notification, error) {
	m.calls++
	m.gotCollegeID = collegeID
	m.gotUserID = userID
	return m.notifications, m.err
}

func (m *mockNotificationHandlerService) MarkRead(
	_ context.Context,
	collegeID int64,
	userID int64,
	notificationID int64,
) (Notification, error) {
	m.calls++
	m.gotCollegeID = collegeID
	m.gotUserID = userID
	m.gotNotificationID = notificationID

	return m.markedNotification, m.markReadErr
}

func (m *mockNotificationHandlerService) UnreadCount(
	_ context.Context,
	collegeID int64,
	userID int64,
) (int64, error) {
	m.calls++
	m.gotCollegeID = collegeID
	m.gotUserID = userID

	return m.unreadCount, m.unreadCountErr
}

func authenticatedRequest(
	method string,
	path string,
) *http.Request {
	req := httptest.NewRequest(method, path, nil)

	claims := &auth.AccessClaims{
		UserID:    42,
		CollegeID: 7,
		Role:      "student",
	}

	ctx := context.WithValue(
		req.Context(),
		notificationTestClaimsContextKey{},
		claims,
	)

	return req.WithContext(ctx)
}

type notificationTestClaimsContextKey struct{}

// withClaims attaches claims using the same context key as auth middleware.
// Tests use the middleware to establish claims instead of accessing private keys.
func TestNewHandler_List(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		service := &mockNotificationHandlerService{
			notifications: []Notification{},
		}
		handler := NewHandler(service)

		tokenService := auth.NewTokenService(
			"test-secret-that-is-at-least-32-bytes",
			time.Minute,
		)
		token, err := tokenService.Generate(auth.User{
			ID:        42,
			CollegeID: 7,
			Role:      "student",
		})
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		recorder := httptest.NewRecorder()
		auth.AuthMiddleware(tokenService, handler).ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s",
				recorder.Code, http.StatusOK, recorder.Body.String())
		}

		if got := recorder.Header().Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}

		var got []Notification
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("notification count = %d, want 0", len(got))
		}

		if service.calls != 1 {
			t.Errorf("List calls = %d, want 1", service.calls)
		}
		if service.gotCollegeID != 7 || service.gotUserID != 42 {
			t.Errorf(
				"List arguments = college %d, user %d; want college 7, user 42",
				service.gotCollegeID,
				service.gotUserID,
			)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		service := &mockNotificationHandlerService{}
		handler := NewHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d",
				recorder.Code, http.StatusUnauthorized)
		}
		if service.calls != 0 {
			t.Errorf("List called %d times, want 0", service.calls)
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		service := &mockNotificationHandlerService{err: ErrInvalidInput}
		handler := NewHandler(service)

		tokenService := auth.NewTokenService(
			"test-secret-that-is-at-least-32-bytes",
			time.Minute,
		)
		token, err := tokenService.Generate(auth.User{
			ID: 42, CollegeID: 7, Role: "student",
		})
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()

		auth.AuthMiddleware(tokenService, handler).ServeHTTP(recorder, req)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body=%s",
				recorder.Code, http.StatusBadRequest, recorder.Body.String())
		}
	})

	t.Run("internal error", func(t *testing.T) {
		service := &mockNotificationHandlerService{
			err: errors.New("database unavailable"),
		}
		handler := NewHandler(service)

		tokenService := auth.NewTokenService(
			"test-secret-that-is-at-least-32-bytes",
			time.Minute,
		)
		token, err := tokenService.Generate(auth.User{
			ID: 42, CollegeID: 7, Role: "student",
		})
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()

		auth.AuthMiddleware(tokenService, handler).ServeHTTP(recorder, req)

		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d",
				recorder.Code, http.StatusInternalServerError)
		}
	})
}

func TestNewHandler_MethodNotAllowed(t *testing.T) {
	handler := NewHandler(&mockNotificationHandlerService{})

	req := httptest.NewRequest(http.MethodPost, "/notifications", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d",
			recorder.Code, http.StatusMethodNotAllowed)
	}
}

func TestNewHandler_MarkRead(t *testing.T) {
	newAuthenticatedHandler := func(
		t *testing.T,
		service *mockNotificationHandlerService,
	) http.Handler {
		t.Helper()

		tokenService := auth.NewTokenService(
			"test-secret-that-is-at-least-32-bytes",
			time.Minute,
		)

		token, err := tokenService.Generate(auth.User{
			ID:        42,
			CollegeID: 7,
			Role:      "student",
		})
		if err != nil {
			t.Fatal(err)
		}

		handler := NewHandler(service)
		protected := auth.AuthMiddleware(tokenService, handler)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("Authorization", "Bearer "+token)
			protected.ServeHTTP(w, r)
		})
	}

	t.Run("success", func(t *testing.T) {
		service := &mockNotificationHandlerService{
			markedNotification: Notification{
				ID:          15,
				RecipientID: 42,
				GroupID:     3,
				Type:        "subgroup_created",
				Message:     "A subgroup was created",
				IsRead:      true,
			},
		}

		handler := newAuthenticatedHandler(t, service)
		req := httptest.NewRequest(
			http.MethodPatch,
			"/notifications/15/read",
			nil,
		)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf(
				"status = %d, want %d; body=%s",
				recorder.Code,
				http.StatusOK,
				recorder.Body.String(),
			)
		}

		if service.gotNotificationID != 15 {
			t.Errorf(
				"notification ID = %d, want 15",
				service.gotNotificationID,
			)
		}
		if service.gotCollegeID != 7 || service.gotUserID != 42 {
			t.Errorf(
				"service received college=%d user=%d; want college=7 user=42",
				service.gotCollegeID,
				service.gotUserID,
			)
		}

		var got Notification
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.ID != 15 || !got.IsRead {
			t.Errorf("response = %+v, want notification 15 marked read", got)
		}
	})

	t.Run("invalid notification ID", func(t *testing.T) {
		service := &mockNotificationHandlerService{}
		handler := newAuthenticatedHandler(t, service)

		req := httptest.NewRequest(
			http.MethodPatch,
			"/notifications/abc/read",
			nil,
		)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf(
				"status = %d, want %d",
				recorder.Code,
				http.StatusBadRequest,
			)
		}
		if service.calls != 0 {
			t.Errorf("service calls = %d, want 0", service.calls)
		}
	})

	t.Run("notification not found", func(t *testing.T) {
		service := &mockNotificationHandlerService{
			markReadErr: ErrNotificationNotFound,
		}
		handler := newAuthenticatedHandler(t, service)

		req := httptest.NewRequest(
			http.MethodPatch,
			"/notifications/999/read",
			nil,
		)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusNotFound {
			t.Fatalf(
				"status = %d, want %d",
				recorder.Code,
				http.StatusNotFound,
			)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		service := &mockNotificationHandlerService{
			markReadErr: errors.New("database unavailable"),
		}
		handler := newAuthenticatedHandler(t, service)

		req := httptest.NewRequest(
			http.MethodPatch,
			"/notifications/15/read",
			nil,
		)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf(
				"status = %d, want %d",
				recorder.Code,
				http.StatusInternalServerError,
			)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		service := &mockNotificationHandlerService{}
		handler := NewHandler(service)

		req := httptest.NewRequest(
			http.MethodPatch,
			"/notifications/15/read",
			nil,
		)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf(
				"status = %d, want %d",
				recorder.Code,
				http.StatusUnauthorized,
			)
		}
		if service.calls != 0 {
			t.Errorf("service calls = %d, want 0", service.calls)
		}
	})
}

func TestNewHandler_UnreadCount(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		service := &mockNotificationHandlerService{
			unreadCount: 5,
		}

		tokenService := auth.NewTokenService(
			"test-secret-that-is-at-least-32-bytes",
			time.Minute,
		)
		token, err := tokenService.Generate(auth.User{
			ID:        42,
			CollegeID: 7,
			Role:      "student",
		})
		if err != nil {
			t.Fatal(err)
		}

		handler := auth.AuthMiddleware(tokenService, NewHandler(service))
		req := httptest.NewRequest(
			http.MethodGet,
			"/notifications/unread-count",
			nil,
		)
		req.Header.Set("Authorization", "Bearer "+token)

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf(
				"status = %d, want %d; body=%s",
				recorder.Code,
				http.StatusOK,
				recorder.Body.String(),
			)
		}

		var got map[string]int64
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got["unread_count"] != 5 {
			t.Errorf("unread_count = %d, want 5", got["unread_count"])
		}
		if service.gotCollegeID != 7 || service.gotUserID != 42 {
			t.Errorf(
				"service received college=%d user=%d; want college=7 user=42",
				service.gotCollegeID,
				service.gotUserID,
			)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		service := &mockNotificationHandlerService{}
		handler := NewHandler(service)

		req := httptest.NewRequest(
			http.MethodGet,
			"/notifications/unread-count",
			nil,
		)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf(
				"status = %d, want %d",
				recorder.Code,
				http.StatusUnauthorized,
			)
		}
		if service.calls != 0 {
			t.Errorf("service calls = %d, want 0", service.calls)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		service := &mockNotificationHandlerService{
			unreadCountErr: errors.New("database unavailable"),
		}

		tokenService := auth.NewTokenService(
			"test-secret-that-is-at-least-32-bytes",
			time.Minute,
		)
		token, err := tokenService.Generate(auth.User{
			ID:        42,
			CollegeID: 7,
			Role:      "student",
		})
		if err != nil {
			t.Fatal(err)
		}

		handler := auth.AuthMiddleware(tokenService, NewHandler(service))
		req := httptest.NewRequest(
			http.MethodGet,
			"/notifications/unread-count",
			nil,
		)
		req.Header.Set("Authorization", "Bearer "+token)

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf(
				"status = %d, want %d",
				recorder.Code,
				http.StatusInternalServerError,
			)
		}
	})
}
