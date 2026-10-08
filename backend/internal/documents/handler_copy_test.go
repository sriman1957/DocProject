package documents

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"docproject/backend/internal/auth"
)

type copyHandlerTestService struct {
	copyFn func(
		ctx context.Context,
		collegeID int64,
		studentID int64,
		documentID int64,
		groupID int64,
		subgroupID int64,
	) (UploadedDocument, error)
}

func (s *copyHandlerTestService) ListPersonalVault(
	context.Context,
	int64,
	int64,
) ([]Document, error) {
	return nil, nil
}

func (s *copyHandlerTestService) GetPersonalVaultDocument(
	context.Context,
	int64,
	int64,
	int64,
) (Document, error) {
	return Document{}, nil
}

func (s *copyHandlerTestService) CreatePersonalVaultDocument(
	context.Context,
	int64,
	int64,
	string,
	[]byte,
) (UploadedDocument, error) {
	return UploadedDocument{}, nil
}

func (s *copyHandlerTestService) PreviewPersonalVaultDocument(
	context.Context,
	int64,
	int64,
	int64,
) (io.ReadCloser, Document, error) {
	return nil, Document{}, nil
}

func (s *copyHandlerTestService) CopyPersonalVaultDocument(
	ctx context.Context,
	collegeID int64,
	studentID int64,
	documentID int64,
	groupID int64,
	subgroupID int64,
) (UploadedDocument, error) {
	if s.copyFn == nil {
		return UploadedDocument{}, errors.New(
			"copy function not configured",
		)
	}

	return s.copyFn(
		ctx,
		collegeID,
		studentID,
		documentID,
		groupID,
		subgroupID,
	)
}

func newCopyHandlerRequest(
	t *testing.T,
	handler http.Handler,
	method string,
	path string,
	role string,
	body string,
	authenticated bool,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(
		method,
		path,
		strings.NewReader(body),
	)

	if authenticated {
		tokens := newDocumentTestTokenService()

		token, err := tokens.Generate(auth.User{
			ID:        42,
			CollegeID: 7,
			Role:      role,
		})
		if err != nil {
			t.Fatalf("generate token: %v", err)
		}

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

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	return recorder
}

func TestCopyPersonalVaultDocumentSuccess(t *testing.T) {
	t.Parallel()

	service := &copyHandlerTestService{
		copyFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			documentID int64,
			groupID int64,
			subgroupID int64,
		) (UploadedDocument, error) {
			if collegeID != 7 {
				t.Errorf("expected college ID 7, got %d", collegeID)
			}
			if studentID != 42 {
				t.Errorf("expected student ID 42, got %d", studentID)
			}
			if documentID != 100 {
				t.Errorf("expected document ID 100, got %d", documentID)
			}
			if groupID != 5 {
				t.Errorf("expected group ID 5, got %d", groupID)
			}
			if subgroupID != 9 {
				t.Errorf("expected subgroup ID 9, got %d", subgroupID)
			}

			return UploadedDocument{
				ID:               200,
				OriginalFilename: "degree.pdf",
				MIMEType:         "application/pdf",
				FileSizeBytes:    1024,
				SHA256:           strings.Repeat("a", 64),
			}, nil
		},
	}

	handler := NewHandler(service)

	recorder := newCopyHandlerRequest(
		t,
		handler,
		http.MethodPost,
		"/documents/100/copy",
		"student",
		`{"group_id":5,"subgroup_id":9}`,
		true,
	)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected status 201, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response UploadedDocument
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.ID != 200 {
		t.Errorf("expected copied document ID 200, got %d", response.ID)
	}
	if response.OriginalFilename != "degree.pdf" {
		t.Errorf("unexpected filename: %q", response.OriginalFilename)
	}
	if response.MIMEType != "application/pdf" {
		t.Errorf("unexpected MIME type: %q", response.MIMEType)
	}
}

func TestCopyPersonalVaultDocumentUnauthorized(t *testing.T) {
	t.Parallel()

	service := &copyHandlerTestService{}
	handler := NewHandler(service)

	recorder := newCopyHandlerRequest(
		t,
		handler,
		http.MethodPost,
		"/documents/100/copy",
		"",
		`{"group_id":5,"subgroup_id":9}`,
		false,
	)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status 401, got %d",
			recorder.Code,
		)
	}
}

func TestCopyPersonalVaultDocumentNonStudentForbidden(t *testing.T) {
	t.Parallel()

	service := &copyHandlerTestService{}
	handler := NewHandler(service)

	recorder := newCopyHandlerRequest(
		t,
		handler,
		http.MethodPost,
		"/documents/100/copy",
		"faculty",
		`{"group_id":5,"subgroup_id":9}`,
		true,
	)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status 403, got %d",
			recorder.Code,
		)
	}
}

func TestCopyPersonalVaultDocumentInvalidDocumentID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
	}{
		{name: "non-numeric ID", path: "/documents/abc/copy"},
		{name: "zero ID", path: "/documents/0/copy"},
		{name: "negative ID", path: "/documents/-1/copy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := NewHandler(&copyHandlerTestService{})

			recorder := newCopyHandlerRequest(
				t,
				handler,
				http.MethodPost,
				tt.path,
				"student",
				`{"group_id":5,"subgroup_id":9}`,
				true,
			)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf(
					"expected status 400, got %d. Body: %s",
					recorder.Code,
					recorder.Body.String(),
				)
			}
		})
	}
}

func TestCopyPersonalVaultDocumentInvalidRequestBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "malformed JSON", body: `{"group_id":`},
		{name: "missing group", body: `{"subgroup_id":9}`},
		{name: "missing subgroup", body: `{"group_id":5}`},
		{name: "zero IDs", body: `{"group_id":0,"subgroup_id":0}`},
		{name: "unknown field", body: `{"group_id":5,"subgroup_id":9,"extra":true}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := NewHandler(&copyHandlerTestService{})

			recorder := newCopyHandlerRequest(
				t,
				handler,
				http.MethodPost,
				"/documents/100/copy",
				"student",
				tt.body,
				true,
			)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf(
					"expected status 400, got %d. Body: %s",
					recorder.Code,
					recorder.Body.String(),
				)
			}
		})
	}
}

func TestCopyPersonalVaultDocumentNotFound(t *testing.T) {
	t.Parallel()

	service := &copyHandlerTestService{
		copyFn: func(
			context.Context,
			int64,
			int64,
			int64,
			int64,
			int64,
		) (UploadedDocument, error) {
			return UploadedDocument{}, ErrDocumentNotFound
		},
	}

	handler := NewHandler(service)

	recorder := newCopyHandlerRequest(
		t,
		handler,
		http.MethodPost,
		"/documents/100/copy",
		"student",
		`{"group_id":5,"subgroup_id":9}`,
		true,
	)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status 404, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func TestCopyPersonalVaultDocumentForbiddenSubgroup(t *testing.T) {
	t.Parallel()

	service := &copyHandlerTestService{
		copyFn: func(
			context.Context,
			int64,
			int64,
			int64,
			int64,
			int64,
		) (UploadedDocument, error) {
			return UploadedDocument{}, ErrForbidden
		},
	}

	handler := NewHandler(service)

	recorder := newCopyHandlerRequest(
		t,
		handler,
		http.MethodPost,
		"/documents/100/copy",
		"student",
		`{"group_id":5,"subgroup_id":9}`,
		true,
	)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status 403, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func TestCopyPersonalVaultDocumentServiceUnavailable(t *testing.T) {
	t.Parallel()

	// This service implements the existing ServiceInterface,
	// but deliberately does not implement CopyService.
	service := &handlerTestService{}
	handler := NewHandler(service)

	recorder := newCopyHandlerRequest(
		t,
		handler,
		http.MethodPost,
		"/documents/100/copy",
		"student",
		`{"group_id":5,"subgroup_id":9}`,
		true,
	)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status 500, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}
}
