package documents

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"docproject/backend/internal/auth"
)

type ListService interface {
	ListPersonalVault(
		ctx context.Context,
		collegeID int64,
		studentID int64,
	) ([]Document, error)
}

type UploadService interface {
	CreatePersonalVaultDocument(
		ctx context.Context,
		collegeID int64,
		studentID int64,
		originalFilename string,
		data []byte,
	) (UploadedDocument, error)
}

type ServiceInterface interface {
	ListService
	UploadService
}

type Handler struct {
	service ServiceInterface
}

func NewHandler(service ServiceInterface) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listPersonalVault(w, r)

	case http.MethodPost:
		h.uploadPersonalVaultDocument(w, r)

	default:
		w.Header().Set(
			"Allow",
			http.MethodGet+", "+http.MethodPost,
		)

		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
	}
}

func (h *Handler) listPersonalVault(
	w http.ResponseWriter,
	r *http.Request,
) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		http.Error(
			w,
			"unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	if claims.Role != "student" {
		http.Error(
			w,
			"forbidden",
			http.StatusForbidden,
		)
		return
	}

	documents, err := h.service.ListPersonalVault(
		r.Context(),
		claims.CollegeID,
		claims.UserID,
	)
	if err != nil {
		writeDocumentServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, documents)
}

func (h *Handler) uploadPersonalVaultDocument(
	w http.ResponseWriter,
	r *http.Request,
) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		http.Error(
			w,
			"unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	if claims.Role != "student" {
		http.Error(
			w,
			"forbidden",
			http.StatusForbidden,
		)
		return
	}

	if !strings.HasPrefix(
		strings.ToLower(r.Header.Get("Content-Type")),
		"multipart/form-data;",
	) {
		http.Error(
			w,
			"content type must be multipart/form-data",
			http.StatusBadRequest,
		)
		return
	}

	// Add a small amount of multipart overhead to the body limit.
	// The actual file is still limited to MaxUploadSize by ValidateUpload.
	const multipartOverhead = 1024 * 1024

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		MaxUploadSize+multipartOverhead,
	)

	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		var maxBytesError *http.MaxBytesError

		if errors.As(err, &maxBytesError) {
			http.Error(
				w,
				"file too large",
				http.StatusRequestEntityTooLarge,
			)
			return
		}

		http.Error(
			w,
			"invalid multipart form",
			http.StatusBadRequest,
		)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			http.Error(
				w,
				"file is required",
				http.StatusBadRequest,
			)
			return
		}

		http.Error(
			w,
			"invalid file upload",
			http.StatusBadRequest,
		)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(
		io.LimitReader(file, MaxUploadSize+1),
	)
	if err != nil {
		http.Error(
			w,
			"failed to read uploaded file",
			http.StatusBadRequest,
		)
		return
	}

	if int64(len(data)) > MaxUploadSize {
		http.Error(
			w,
			"file too large",
			http.StatusRequestEntityTooLarge,
		)
		return
	}

	document, err := h.service.CreatePersonalVaultDocument(
		r.Context(),
		claims.CollegeID,
		claims.UserID,
		header.Filename,
		data,
	)
	if err != nil {
		writeDocumentServiceError(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		document,
	)
}

func writeDocumentServiceError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		http.Error(
			w,
			"invalid input",
			http.StatusBadRequest,
		)

	case errors.Is(err, ErrFileTooLarge):
		http.Error(
			w,
			"file too large",
			http.StatusRequestEntityTooLarge,
		)

	case errors.Is(err, ErrUnsupportedType):
		http.Error(
			w,
			"unsupported file type",
			http.StatusBadRequest,
		)

	case errors.Is(err, ErrInvalidFile):
		http.Error(
			w,
			"invalid file",
			http.StatusBadRequest,
		)

	case errors.Is(err, ErrInvalidDOCX):
		http.Error(
			w,
			"invalid DOCX file",
			http.StatusBadRequest,
		)

	case errors.Is(err, ErrForbidden):
		http.Error(
			w,
			"forbidden",
			http.StatusForbidden,
		)

	default:
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
	}
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	value any,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		return
	}
}
