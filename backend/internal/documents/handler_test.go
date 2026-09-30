package documents

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
	listFn func(
		ctx context.Context,
		collegeID int64,
		studentID int64,
	) ([]Document, error)
}

func (s *handlerTestService) ListPersonalVault(
	ctx context.Context,
	collegeID int64,
	studentID int64,
) ([]Document, error) {
	return s.listFn(ctx, collegeID, studentID)
}

func newDocumentTestTokenService() *auth.TokenService {
	return auth.NewTokenService(
		"01234567890123456789012345678901",
		15*time.Minute,
	)
}

func makeDocumentAuthenticatedRequest(
	t *testing.T,
	handler http.Handler,
	method string,
	path string,
	role string,
) *httptest.ResponseRecorder {
	t.Helper()

	tokens := newDocumentTestTokenService()

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
		strings.NewReader(""),
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

func TestListPersonalVaultHandlerSuccess(t *testing.T) {
	t.Parallel()

	uploadedAt := time.Date(
		2026,
		9,
		29,
		10,
		0,
		0,
		0,
		time.UTC,
	)

	service := &handlerTestService{
		listFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
		) ([]Document, error) {
			if collegeID != 7 {
				t.Errorf(
					"expected college ID 7, got %d",
					collegeID,
				)
			}

			if studentID != 42 {
				t.Errorf(
					"expected student ID 42, got %d",
					studentID,
				)
			}

			return []Document{
				{
					ID:               100,
					OriginalFilename: "degree.pdf",
					MIMEType:         "application/pdf",
					FileSizeBytes:    245678,
					SHA256:           "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					UploadedAt:       uploadedAt,
				},
			}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeDocumentAuthenticatedRequest(
		t,
		handler,
		http.MethodGet,
		"/documents",
		"student",
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status 200, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response []Document

	if err := json.NewDecoder(
		recorder.Body,
	).Decode(&response); err != nil {
		t.Fatalf(
			"decode response: %v",
			err,
		)
	}

	if len(response) != 1 {
		t.Fatalf(
			"expected 1 document, got %d",
			len(response),
		)
	}

	if response[0].ID != 100 {
		t.Errorf(
			"expected document ID 100, got %d",
			response[0].ID,
		)
	}

	if response[0].OriginalFilename != "degree.pdf" {
		t.Errorf(
			"expected filename degree.pdf, got %s",
			response[0].OriginalFilename,
		)
	}

	if response[0].MIMEType != "application/pdf" {
		t.Errorf(
			"expected MIME type application/pdf, got %s",
			response[0].MIMEType,
		)
	}

	if response[0].FileSizeBytes != 245678 {
		t.Errorf(
			"expected file size 245678, got %d",
			response[0].FileSizeBytes,
		)
	}

	if !response[0].UploadedAt.Equal(uploadedAt) {
		t.Errorf(
			"unexpected uploaded_at: %s",
			response[0].UploadedAt,
		)
	}
}

func TestListPersonalVaultHandlerEmpty(t *testing.T) {
	t.Parallel()

	service := &handlerTestService{
		listFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
		) ([]Document, error) {
			return []Document{}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeDocumentAuthenticatedRequest(
		t,
		handler,
		http.MethodGet,
		"/documents",
		"student",
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status 200, got %d",
			recorder.Code,
		)
	}

	var response []Document

	if err := json.NewDecoder(
		recorder.Body,
	).Decode(&response); err != nil {
		t.Fatalf(
			"decode response: %v",
			err,
		)
	}

	if response == nil {
		t.Fatal("expected empty JSON array, got null")
	}

	if len(response) != 0 {
		t.Fatalf(
			"expected 0 documents, got %d",
			len(response),
		)
	}
}

func TestListPersonalVaultHandlerUnauthorized(t *testing.T) {
	t.Parallel()

	service := &handlerTestService{
		listFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
		) ([]Document, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/documents",
		nil,
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

func TestListPersonalVaultHandlerFacultyForbidden(t *testing.T) {
	t.Parallel()

	service := &handlerTestService{
		listFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
		) ([]Document, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeDocumentAuthenticatedRequest(
		t,
		handler,
		http.MethodGet,
		"/documents",
		"faculty",
	)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status 403, got %d",
			recorder.Code,
		)
	}
}

func TestListPersonalVaultHandlerCollegeAdminForbidden(t *testing.T) {
	t.Parallel()

	service := &handlerTestService{
		listFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
		) ([]Document, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeDocumentAuthenticatedRequest(
		t,
		handler,
		http.MethodGet,
		"/documents",
		"college_admin",
	)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status 403, got %d",
			recorder.Code,
		)
	}
}

func TestListPersonalVaultHandlerServiceError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("database failure")

	service := &handlerTestService{
		listFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
		) ([]Document, error) {
			return nil, expectedErr
		},
	}

	handler := NewHandler(service)

	recorder := makeDocumentAuthenticatedRequest(
		t,
		handler,
		http.MethodGet,
		"/documents",
		"student",
	)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status 500, got %d",
			recorder.Code,
		)
	}
}

func TestListPersonalVaultHandlerForbidden(t *testing.T) {
	t.Parallel()

	service := &handlerTestService{
		listFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
		) ([]Document, error) {
			return nil, ErrForbidden
		},
	}

	handler := NewHandler(service)

	recorder := makeDocumentAuthenticatedRequest(
		t,
		handler,
		http.MethodGet,
		"/documents",
		"student",
	)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status 403, got %d",
			recorder.Code,
		)
	}
}

func TestListPersonalVaultHandlerInvalidInput(t *testing.T) {
	t.Parallel()

	service := &handlerTestService{
		listFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
		) ([]Document, error) {
			return nil, ErrInvalidInput
		},
	}

	handler := NewHandler(service)

	recorder := makeDocumentAuthenticatedRequest(
		t,
		handler,
		http.MethodGet,
		"/documents",
		"student",
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status 400, got %d",
			recorder.Code,
		)
	}
}

func TestListPersonalVaultHandlerMethodNotAllowed(t *testing.T) {
	t.Parallel()

	service := &handlerTestService{
		listFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
		) ([]Document, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeDocumentAuthenticatedRequest(
		t,
		handler,
		http.MethodPost,
		"/documents",
		"student",
	)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"expected status 405, got %d",
			recorder.Code,
		)
	}

	if recorder.Header().Get("Allow") != http.MethodGet {
		t.Fatalf(
			"expected Allow: GET, got %q",
			recorder.Header().Get("Allow"),
		)
	}
}
