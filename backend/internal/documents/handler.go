package documents

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
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

type GetService interface {
	GetPersonalVaultDocument(
		ctx context.Context,
		collegeID int64,
		studentID int64,
		documentID int64,
	) (Document, error)
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

type PreviewService interface {
	PreviewPersonalVaultDocument(
		ctx context.Context,
		collegeID int64,
		studentID int64,
		documentID int64,
	) (io.ReadCloser, Document, error)
}

type ServiceInterface interface {
	ListService
	GetService
	UploadService
	PreviewService
}

type Handler struct {
	service ServiceInterface
}

func NewHandler(service ServiceInterface) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	switch {
	case r.Method == http.MethodGet &&
		r.URL.Path == "/documents":
		h.listPersonalVault(w, r)

	case r.Method == http.MethodGet &&
		strings.HasPrefix(r.URL.Path, "/documents/") &&
		strings.HasSuffix(r.URL.Path, "/preview"):
		h.previewPersonalVaultDocument(w, r)

	case r.Method == http.MethodGet &&
		strings.HasPrefix(r.URL.Path, "/documents/"):
		h.getPersonalVaultDocument(w, r)

	case r.Method == http.MethodPost &&
		r.URL.Path == "/documents":
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

func (h *Handler) getPersonalVaultDocument(
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

	const prefix = "/documents/"

	documentIDText := strings.TrimPrefix(
		r.URL.Path,
		prefix,
	)

	documentID, err := strconv.ParseInt(
		documentIDText,
		10,
		64,
	)
	if err != nil || documentID <= 0 {
		http.Error(
			w,
			"invalid document ID",
			http.StatusBadRequest,
		)
		return
	}

	document, err := h.service.GetPersonalVaultDocument(
		r.Context(),
		claims.CollegeID,
		claims.UserID,
		documentID,
	)
	if err != nil {
		writeDocumentServiceError(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		document,
	)
}

func (h *Handler) previewPersonalVaultDocument(
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

	const prefix = "/documents/"
	const suffix = "/preview"

	documentIDText := strings.TrimSuffix(
		strings.TrimPrefix(r.URL.Path, prefix),
		suffix,
	)

	documentID, err := strconv.ParseInt(
		documentIDText,
		10,
		64,
	)
	if err != nil || documentID <= 0 {
		http.Error(
			w,
			"invalid document ID",
			http.StatusBadRequest,
		)
		return
	}

	file, document, err := h.service.PreviewPersonalVaultDocument(
		r.Context(),
		claims.CollegeID,
		claims.UserID,
		documentID,
	)
	if err != nil {
		writeDocumentServiceError(w, err)
		return
	}
	defer file.Close()

	w.Header().Set(
		"Content-Type",
		document.MIMEType,
	)

	w.Header().Set(
		"Content-Length",
		strconv.FormatInt(document.FileSizeBytes, 10),
	)

	w.Header().Set(
		"Content-Disposition",
		"inline; filename=\""+
			strings.ReplaceAll(document.OriginalFilename, "\"", "")+
			"\"",
	)

	w.WriteHeader(http.StatusOK)

	_, _ = io.Copy(w, file)
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

	case errors.Is(err, ErrDocumentNotFound):
		http.Error(
			w,
			"document not found",
			http.StatusNotFound,
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
