package documents

import (
	"bytes"
	"context"
	"io"
	"strconv"
	"encoding/json"
	"errors"
	"mime/multipart"
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

	getFn func(
		ctx context.Context,
		collegeID int64,
		studentID int64,
		documentID int64,
	) (Document, error)

	uploadFn func(
		ctx context.Context,
		collegeID int64,
		studentID int64,
		originalFilename string,
		data []byte,
	) (UploadedDocument, error)

	previewFn func(
		ctx context.Context,
		collegeID int64,
		studentID int64,
		documentID int64,
	) (io.ReadCloser, Document, error)
}

func (s *handlerTestService) ListPersonalVault(
	ctx context.Context,
	collegeID int64,
	studentID int64,
) ([]Document, error) {
	if s.listFn == nil {
		return nil, errors.New(
			"list function not configured",
		)
	}

	return s.listFn(
		ctx,
		collegeID,
		studentID,
	)
}

func (s *handlerTestService) GetPersonalVaultDocument(
	ctx context.Context,
	collegeID int64,
	studentID int64,
	documentID int64,
) (Document, error) {
	if s.getFn == nil {
		return Document{}, errors.New(
			"get function not configured",
		)
	}

	return s.getFn(
		ctx,
		collegeID,
		studentID,
		documentID,
	)
}

func (s *handlerTestService) CreatePersonalVaultDocument(
	ctx context.Context,
	collegeID int64,
	studentID int64,
	originalFilename string,
	data []byte,
) (UploadedDocument, error) {
	if s.uploadFn == nil {
		return UploadedDocument{}, errors.New(
			"upload function not configured",
		)
	}

	return s.uploadFn(
		ctx,
		collegeID,
		studentID,
		originalFilename,
		data,
	)
}

func (s *handlerTestService) PreviewPersonalVaultDocument(
	ctx context.Context,
	collegeID int64,
	studentID int64,
	documentID int64,
) (io.ReadCloser, Document, error) {
	if s.previewFn == nil {
		return nil, Document{}, errors.New(
			"preview function not configured",
		)
	}

	return s.previewFn(
		ctx,
		collegeID,
		studentID,
		documentID,
	)
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

func makeMultipartUploadRequest(
	t *testing.T,
	handler http.Handler,
	role string,
	filename string,
	data []byte,
	contentType string,
) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile(
		"file",
		filename,
	)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}

	if _, err := part.Write(data); err != nil {
		t.Fatalf("write multipart data: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

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
		http.MethodPost,
		"/documents",
		&body,
	)

	if contentType == "" {
		contentType = writer.FormDataContentType()
	}

	req.Header.Set(
		"Content-Type",
		contentType,
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

	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatalf(
			"expected JSON content type, got %q",
			recorder.Header().Get("Content-Type"),
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
		http.MethodPut,
		"/documents",
		"student",
	)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"expected status 405, got %d",
			recorder.Code,
		)
	}

	expectedAllow := http.MethodGet + ", " + http.MethodPost

	if recorder.Header().Get("Allow") != expectedAllow {
		t.Fatalf(
			"expected Allow: %s, got %q",
			expectedAllow,
			recorder.Header().Get("Allow"),
		)
	}
}

func TestGetPersonalVaultDocumentHandlerSuccess(t *testing.T) {
	t.Parallel()

	uploadedAt := time.Date(
		2026,
		9,
		30,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	service := &handlerTestService{
		getFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			documentID int64,
		) (Document, error) {
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

			if documentID != 100 {
				t.Errorf(
					"expected document ID 100, got %d",
					documentID,
				)
			}

			return Document{
				ID:               100,
				OriginalFilename: "certificate.pdf",
				MIMEType:         "application/pdf",
				FileSizeBytes:    1024,
				SHA256:           "abc123",
				UploadedAt:       uploadedAt,
			}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeDocumentAuthenticatedRequest(
		t,
		handler,
		http.MethodGet,
		"/documents/100",
		"student",
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status 200, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatalf(
			"expected JSON content type, got %q",
			recorder.Header().Get("Content-Type"),
		)
	}

	var response Document

	if err := json.NewDecoder(
		recorder.Body,
	).Decode(&response); err != nil {
		t.Fatalf(
			"decode response: %v",
			err,
		)
	}

	if response.ID != 100 {
		t.Errorf(
			"expected document ID 100, got %d",
			response.ID,
		)
	}

	if response.OriginalFilename != "certificate.pdf" {
		t.Errorf(
			"expected filename certificate.pdf, got %s",
			response.OriginalFilename,
		)
	}

	if response.MIMEType != "application/pdf" {
		t.Errorf(
			"expected MIME type application/pdf, got %s",
			response.MIMEType,
		)
	}

	if response.FileSizeBytes != 1024 {
		t.Errorf(
			"expected file size 1024, got %d",
			response.FileSizeBytes,
		)
	}

	if response.SHA256 != "abc123" {
		t.Errorf(
			"expected SHA256 abc123, got %s",
			response.SHA256,
		)
	}

	if !response.UploadedAt.Equal(uploadedAt) {
		t.Errorf(
			"unexpected uploaded_at: %s",
			response.UploadedAt,
		)
	}
}

func TestGetPersonalVaultDocumentHandlerUnauthorized(t *testing.T) {
	t.Parallel()

	service := &handlerTestService{
		getFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			documentID int64,
		) (Document, error) {
			t.Fatal("service should not be called")
			return Document{}, nil
		},
	}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/documents/100",
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

func TestGetPersonalVaultDocumentHandlerFacultyForbidden(t *testing.T) {
	t.Parallel()

	service := &handlerTestService{
		getFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			documentID int64,
		) (Document, error) {
			t.Fatal("service should not be called")
			return Document{}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeDocumentAuthenticatedRequest(
		t,
		handler,
		http.MethodGet,
		"/documents/100",
		"faculty",
	)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status 403, got %d",
			recorder.Code,
		)
	}
}

func TestGetPersonalVaultDocumentHandlerCollegeAdminForbidden(t *testing.T) {
	t.Parallel()

	service := &handlerTestService{
		getFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			documentID int64,
		) (Document, error) {
			t.Fatal("service should not be called")
			return Document{}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeDocumentAuthenticatedRequest(
		t,
		handler,
		http.MethodGet,
		"/documents/100",
		"college_admin",
	)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status 403, got %d",
			recorder.Code,
		)
	}
}

func TestGetPersonalVaultDocumentHandlerInvalidID(t *testing.T) {
	t.Parallel()

	service := &handlerTestService{
		getFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			documentID int64,
		) (Document, error) {
			t.Fatal("service should not be called")
			return Document{}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeDocumentAuthenticatedRequest(
		t,
		handler,
		http.MethodGet,
		"/documents/not-a-number",
		"student",
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status 400, got %d",
			recorder.Code,
		)
	}
}

func TestGetPersonalVaultDocumentHandlerZeroID(t *testing.T) {
	t.Parallel()

	service := &handlerTestService{
		getFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			documentID int64,
		) (Document, error) {
			t.Fatal("service should not be called")
			return Document{}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeDocumentAuthenticatedRequest(
		t,
		handler,
		http.MethodGet,
		"/documents/0",
		"student",
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status 400, got %d",
			recorder.Code,
		)
	}
}

func TestGetPersonalVaultDocumentHandlerNotFound(t *testing.T) {
	t.Parallel()

	service := &handlerTestService{
		getFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			documentID int64,
		) (Document, error) {
			return Document{}, ErrDocumentNotFound
		},
	}

	handler := NewHandler(service)

	recorder := makeDocumentAuthenticatedRequest(
		t,
		handler,
		http.MethodGet,
		"/documents/100",
		"student",
	)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status 404, got %d",
			recorder.Code,
		)
	}
}

func TestGetPersonalVaultDocumentHandlerInvalidInput(t *testing.T) {
	t.Parallel()

	service := &handlerTestService{
		getFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			documentID int64,
		) (Document, error) {
			return Document{}, ErrInvalidInput
		},
	}

	handler := NewHandler(service)

	recorder := makeDocumentAuthenticatedRequest(
		t,
		handler,
		http.MethodGet,
		"/documents/100",
		"student",
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status 400, got %d",
			recorder.Code,
		)
	}
}

func TestGetPersonalVaultDocumentHandlerForbidden(t *testing.T) {
	t.Parallel()

	service := &handlerTestService{
		getFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			documentID int64,
		) (Document, error) {
			return Document{}, ErrForbidden
		},
	}

	handler := NewHandler(service)

	recorder := makeDocumentAuthenticatedRequest(
		t,
		handler,
		http.MethodGet,
		"/documents/100",
		"student",
	)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status 403, got %d",
			recorder.Code,
		)
	}
}

func TestGetPersonalVaultDocumentHandlerServiceError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("database failure")

	service := &handlerTestService{
		getFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			documentID int64,
		) (Document, error) {
			return Document{}, expectedErr
		},
	}

	handler := NewHandler(service)

	recorder := makeDocumentAuthenticatedRequest(
		t,
		handler,
		http.MethodGet,
		"/documents/100",
		"student",
	)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status 500, got %d",
			recorder.Code,
		)
	}
}

func TestUploadPersonalVaultDocumentHandlerSuccess(t *testing.T) {
	t.Parallel()

	data := append(
		[]byte("%PDF-1.7\n"),
		[]byte("certificate")...,
	)

	uploadedAt := time.Date(
		2026,
		9,
		30,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	service := &handlerTestService{
		uploadFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			originalFilename string,
			uploadedData []byte,
		) (UploadedDocument, error) {
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

			if originalFilename != "certificate.pdf" {
				t.Errorf(
					"expected filename certificate.pdf, got %s",
					originalFilename,
				)
			}

			if !bytes.Equal(uploadedData, data) {
				t.Error("uploaded data does not match request data")
			}

			return UploadedDocument{
				ID:               101,
				OriginalFilename: "certificate.pdf",
				MIMEType:         MIMEPDF,
				FileSizeBytes:    int64(len(data)),
				SHA256:           CalculateSHA256(data),
				UploadedAt:       uploadedAt,
			}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeMultipartUploadRequest(
		t,
		handler,
		"student",
		"certificate.pdf",
		data,
		"",
	)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected status 201, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatalf(
			"expected JSON content type, got %q",
			recorder.Header().Get("Content-Type"),
		)
	}

	var response UploadedDocument

	if err := json.NewDecoder(
		recorder.Body,
	).Decode(&response); err != nil {
		t.Fatalf(
			"decode response: %v",
			err,
		)
	}

	if response.ID != 101 {
		t.Errorf(
			"expected document ID 101, got %d",
			response.ID,
		)
	}

	if response.OriginalFilename != "certificate.pdf" {
		t.Errorf(
			"expected filename certificate.pdf, got %s",
			response.OriginalFilename,
		)
	}

	if response.MIMEType != MIMEPDF {
		t.Errorf(
			"expected MIME type %s, got %s",
			MIMEPDF,
			response.MIMEType,
		)
	}

	if response.FileSizeBytes != int64(len(data)) {
		t.Errorf(
			"expected file size %d, got %d",
			len(data),
			response.FileSizeBytes,
		)
	}

	if response.SHA256 != CalculateSHA256(data) {
		t.Errorf(
			"unexpected SHA-256: %s",
			response.SHA256,
		)
	}

	if !response.UploadedAt.Equal(uploadedAt) {
		t.Errorf(
			"unexpected uploaded_at: %s",
			response.UploadedAt,
		)
	}
}

func TestUploadPersonalVaultDocumentHandlerUnauthorized(t *testing.T) {
	t.Parallel()

	service := &handlerTestService{
		uploadFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			originalFilename string,
			data []byte,
		) (UploadedDocument, error) {
			t.Fatal("service should not be called")
			return UploadedDocument{}, nil
		},
	}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
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

func TestUploadPersonalVaultDocumentHandlerFacultyForbidden(t *testing.T) {
	t.Parallel()

	service := &handlerTestService{
		uploadFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			originalFilename string,
			data []byte,
		) (UploadedDocument, error) {
			t.Fatal("service should not be called")
			return UploadedDocument{}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeMultipartUploadRequest(
		t,
		handler,
		"faculty",
		"certificate.pdf",
		[]byte("%PDF-1.7\ncertificate"),
		"",
	)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status 403, got %d",
			recorder.Code,
		)
	}
}

func TestUploadPersonalVaultDocumentHandlerCollegeAdminForbidden(
	t *testing.T,
) {
	t.Parallel()

	service := &handlerTestService{
		uploadFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			originalFilename string,
			data []byte,
		) (UploadedDocument, error) {
			t.Fatal("service should not be called")
			return UploadedDocument{}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeMultipartUploadRequest(
		t,
		handler,
		"college_admin",
		"certificate.pdf",
		[]byte("%PDF-1.7\ncertificate"),
		"",
	)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status 403, got %d",
			recorder.Code,
		)
	}
}

func TestUploadPersonalVaultDocumentHandlerMissingFile(t *testing.T) {
	t.Parallel()

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	if err := writer.Close(); err != nil {
		t.Fatalf(
			"close multipart writer: %v",
			err,
		)
	}

	tokens := newDocumentTestTokenService()

	token, err := tokens.Generate(auth.User{
		ID:        42,
		CollegeID: 7,
		Role:      "student",
	})
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	service := &handlerTestService{
		uploadFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			originalFilename string,
			data []byte,
		) (UploadedDocument, error) {
			t.Fatal("service should not be called")
			return UploadedDocument{}, nil
		},
	}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/documents",
		&body,
	)

	req.Header.Set(
		"Content-Type",
		writer.FormDataContentType(),
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

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status 400, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func TestUploadPersonalVaultDocumentHandlerWrongContentType(
	t *testing.T,
) {
	t.Parallel()

	service := &handlerTestService{
		uploadFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			originalFilename string,
			data []byte,
		) (UploadedDocument, error) {
			t.Fatal("service should not be called")
			return UploadedDocument{}, nil
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

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status 400, got %d",
			recorder.Code,
		)
	}
}

func TestUploadPersonalVaultDocumentHandlerOversized(
	t *testing.T,
) {
	t.Parallel()

	service := &handlerTestService{
		uploadFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			originalFilename string,
			data []byte,
		) (UploadedDocument, error) {
			t.Fatal("service should not be called")
			return UploadedDocument{}, nil
		},
	}

	handler := NewHandler(service)

	data := bytes.Repeat(
		[]byte("A"),
		int(MaxUploadSize)+1,
	)

	recorder := makeMultipartUploadRequest(
		t,
		handler,
		"student",
		"large.pdf",
		data,
		"",
	)

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf(
			"expected status 413, got %d",
			recorder.Code,
		)
	}
}

func TestUploadPersonalVaultDocumentHandlerServiceInvalidInput(
	t *testing.T,
) {
	t.Parallel()

	service := &handlerTestService{
		uploadFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			originalFilename string,
			data []byte,
		) (UploadedDocument, error) {
			return UploadedDocument{}, ErrInvalidInput
		},
	}

	handler := NewHandler(service)

	data := []byte("%PDF-1.7\ncertificate")

	recorder := makeMultipartUploadRequest(
		t,
		handler,
		"student",
		"certificate.pdf",
		data,
		"",
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status 400, got %d",
			recorder.Code,
		)
	}
}

func TestUploadPersonalVaultDocumentHandlerUnsupportedType(
	t *testing.T,
) {
	t.Parallel()

	service := &handlerTestService{
		uploadFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			originalFilename string,
			data []byte,
		) (UploadedDocument, error) {
			return UploadedDocument{}, ErrUnsupportedType
		},
	}

	handler := NewHandler(service)

	recorder := makeMultipartUploadRequest(
		t,
		handler,
		"student",
		"certificate.exe",
		[]byte("not a supported file"),
		"",
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status 400, got %d",
			recorder.Code,
		)
	}
}

func TestUploadPersonalVaultDocumentHandlerInvalidFile(
	t *testing.T,
) {
	t.Parallel()

	service := &handlerTestService{
		uploadFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			originalFilename string,
			data []byte,
		) (UploadedDocument, error) {
			return UploadedDocument{}, ErrInvalidFile
		},
	}

	handler := NewHandler(service)

	recorder := makeMultipartUploadRequest(
		t,
		handler,
		"student",
		"certificate.pdf",
		[]byte("invalid PDF"),
		"",
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status 400, got %d",
			recorder.Code,
		)
	}
}

func TestUploadPersonalVaultDocumentHandlerInvalidDOCX(
	t *testing.T,
) {
	t.Parallel()

	service := &handlerTestService{
		uploadFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			originalFilename string,
			data []byte,
		) (UploadedDocument, error) {
			return UploadedDocument{}, ErrInvalidDOCX
		},
	}

	handler := NewHandler(service)

	recorder := makeMultipartUploadRequest(
		t,
		handler,
		"student",
		"certificate.docx",
		[]byte("invalid DOCX"),
		"",
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status 400, got %d",
			recorder.Code,
		)
	}
}

func TestUploadPersonalVaultDocumentHandlerForbiddenServiceError(
	t *testing.T,
) {
	t.Parallel()

	service := &handlerTestService{
		uploadFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			originalFilename string,
			data []byte,
		) (UploadedDocument, error) {
			return UploadedDocument{}, ErrForbidden
		},
	}

	handler := NewHandler(service)

	data := []byte("%PDF-1.7\ncertificate")

	recorder := makeMultipartUploadRequest(
		t,
		handler,
		"student",
		"certificate.pdf",
		data,
		"",
	)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status 403, got %d",
			recorder.Code,
		)
	}
}

func TestUploadPersonalVaultDocumentHandlerServiceError(
	t *testing.T,
) {
	t.Parallel()

	expectedErr := errors.New("storage failure")

	service := &handlerTestService{
		uploadFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			originalFilename string,
			data []byte,
		) (UploadedDocument, error) {
			return UploadedDocument{}, expectedErr
		},
	}

	handler := NewHandler(service)

	data := []byte("%PDF-1.7\ncertificate")

	recorder := makeMultipartUploadRequest(
		t,
		handler,
		"student",
		"certificate.pdf",
		data,
		"",
	)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status 500, got %d",
			recorder.Code,
		)
	}
}

func TestUploadPersonalVaultDocumentHandlerMethodNotAllowed(
	t *testing.T,
) {
	t.Parallel()

	service := &handlerTestService{
		uploadFn: func(
			ctx context.Context,
			collegeID int64,
			studentID int64,
			originalFilename string,
			data []byte,
		) (UploadedDocument, error) {
			t.Fatal("service should not be called")
			return UploadedDocument{}, nil
		},
	}

	handler := NewHandler(service)

	recorder := makeDocumentAuthenticatedRequest(
		t,
		handler,
		http.MethodDelete,
		"/documents",
		"student",
	)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"expected status 405, got %d",
			recorder.Code,
		)
	}

	expectedAllow := http.MethodGet + ", " + http.MethodPost

	if recorder.Header().Get("Allow") != expectedAllow {
		t.Fatalf(
			"expected Allow: %s, got %q",
			expectedAllow,
			recorder.Header().Get("Allow"),
		)
	}
}

func TestPreviewPersonalVaultDocumentHandlerSuccess(t *testing.T) {
	t.Parallel()

	data := []byte("%PDF-1.7\ncertificate contents")

	service := &handlerTestService{previewFn: func(ctx context.Context, collegeID int64, studentID int64, documentID int64) (io.ReadCloser, Document, error) {
		if collegeID != 7 || studentID != 42 || documentID != 100 { t.Fatalf("unexpected identity: %d/%d/%d", collegeID, studentID, documentID) }
		return io.NopCloser(bytes.NewReader(data)), Document{ID:100, OriginalFilename:"certificate.pdf", MIMEType:"application/pdf", FileSizeBytes:int64(len(data))}, nil
	}}

	handler := NewHandler(service)
	recorder := makeDocumentAuthenticatedRequest(t, handler, http.MethodGet, "/documents/100/preview", "student")
	if recorder.Code != http.StatusOK { t.Fatalf("expected status 200, got %d. Body: %s", recorder.Code, recorder.Body.String()) }
	if recorder.Header().Get("Content-Type") != "application/pdf" { t.Fatalf("unexpected Content-Type: %q", recorder.Header().Get("Content-Type")) }
	if recorder.Header().Get("Content-Length") != strconv.Itoa(len(data)) { t.Fatalf("unexpected Content-Length: %q", recorder.Header().Get("Content-Length")) }
	if recorder.Header().Get("Content-Disposition") != "inline; filename=\"certificate.pdf\"" { t.Fatalf("unexpected Content-Disposition: %q", recorder.Header().Get("Content-Disposition")) }
	if !bytes.Equal(recorder.Body.Bytes(), data) { t.Fatal("preview response body does not match document") }
}

func TestPreviewPersonalVaultDocumentHandlerUnauthorized(t *testing.T) {
	t.Parallel()
	service := &handlerTestService{previewFn: func(ctx context.Context, collegeID int64, studentID int64, documentID int64) (io.ReadCloser, Document, error) { t.Fatal("service should not be called"); return nil, Document{}, nil }}
	handler := NewHandler(service)
	req := httptest.NewRequest(http.MethodGet, "/documents/100/preview", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnauthorized { t.Fatalf("expected status 401, got %d", recorder.Code) }
}

func TestPreviewPersonalVaultDocumentHandlerFacultyForbidden(t *testing.T) {
	t.Parallel()
	service := &handlerTestService{previewFn: func(ctx context.Context, collegeID int64, studentID int64, documentID int64) (io.ReadCloser, Document, error) { t.Fatal("service should not be called"); return nil, Document{}, nil }}
	handler := NewHandler(service)
	recorder := makeDocumentAuthenticatedRequest(t, handler, http.MethodGet, "/documents/100/preview", "faculty")
	if recorder.Code != http.StatusForbidden { t.Fatalf("expected status 403, got %d", recorder.Code) }
}

func TestPreviewPersonalVaultDocumentHandlerCollegeAdminForbidden(t *testing.T) {
	t.Parallel()
	service := &handlerTestService{previewFn: func(ctx context.Context, collegeID int64, studentID int64, documentID int64) (io.ReadCloser, Document, error) { t.Fatal("service should not be called"); return nil, Document{}, nil }}
	handler := NewHandler(service)
	recorder := makeDocumentAuthenticatedRequest(t, handler, http.MethodGet, "/documents/100/preview", "college_admin")
	if recorder.Code != http.StatusForbidden { t.Fatalf("expected status 403, got %d", recorder.Code) }
}

func TestPreviewPersonalVaultDocumentHandlerInvalidID(t *testing.T) {
	t.Parallel()
	service := &handlerTestService{previewFn: func(ctx context.Context, collegeID int64, studentID int64, documentID int64) (io.ReadCloser, Document, error) { t.Fatal("service should not be called"); return nil, Document{}, nil }}
	handler := NewHandler(service)
	recorder := makeDocumentAuthenticatedRequest(t, handler, http.MethodGet, "/documents/not-a-number/preview", "student")
	if recorder.Code != http.StatusBadRequest { t.Fatalf("expected status 400, got %d", recorder.Code) }
}

func TestPreviewPersonalVaultDocumentHandlerNotFound(t *testing.T) {
	t.Parallel()
	service := &handlerTestService{previewFn: func(ctx context.Context, collegeID int64, studentID int64, documentID int64) (io.ReadCloser, Document, error) { return nil, Document{}, ErrDocumentNotFound }}
	handler := NewHandler(service)
	recorder := makeDocumentAuthenticatedRequest(t, handler, http.MethodGet, "/documents/100/preview", "student")
	if recorder.Code != http.StatusNotFound { t.Fatalf("expected status 404, got %d", recorder.Code) }
}

func TestPreviewPersonalVaultDocumentHandlerServiceError(t *testing.T) {
	t.Parallel()
	expectedErr := errors.New("preview failure")
	service := &handlerTestService{previewFn: func(ctx context.Context, collegeID int64, studentID int64, documentID int64) (io.ReadCloser, Document, error) { return nil, Document{}, expectedErr }}
	handler := NewHandler(service)
	recorder := makeDocumentAuthenticatedRequest(t, handler, http.MethodGet, "/documents/100/preview", "student")
	if recorder.Code != http.StatusInternalServerError { t.Fatalf("expected status 500, got %d", recorder.Code) }
}
