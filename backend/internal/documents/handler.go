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

// CopyService defines the operation for copying a Personal Vault
// document into a subgroup.
type CopyService interface {
	CopyPersonalVaultDocument(
		ctx context.Context,
		collegeID int64,
		studentID int64,
		documentID int64,
		groupID int64,
		subgroupID int64,
	) (UploadedDocument, error)
}

type ServiceInterface interface {
	ListService
	GetService
	UploadService
	PreviewService
}

// SubgroupDocumentListService defines the operation for listing
// documents stored in a subgroup.
type SubgroupDocumentListService interface {
	ListSubgroupDocuments(
		ctx context.Context,
		collegeID int64,
		actorID int64,
		actorRole string,
		groupID int64,
		subgroupID int64,
	) ([]Document, error)
}

type SubgroupDocumentPreviewService interface {
	PreviewSubgroupDocument(
		ctx context.Context,
		collegeID int64,
		actorID int64,
		actorRole string,
		groupID int64,
		subgroupID int64,
		documentID int64,
	) (io.ReadCloser, Document, error)
}

type SubgroupDocumentDownloadService interface {
	DownloadSubgroupDocument(
		ctx context.Context,
		collegeID int64,
		actorID int64,
		actorRole string,
		groupID int64,
		subgroupID int64,
		documentID int64,
	) (io.ReadCloser, Document, error)
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
		r.PathValue("group_id") != "" &&
		r.PathValue("subgroup_id") != "" &&
		strings.HasSuffix(r.URL.Path, "/documents"):
		h.ListSubgroupDocuments(w, r)

	case r.Method == http.MethodGet &&
		r.URL.Path == "/documents":
		h.listPersonalVault(w, r)

	case r.Method == http.MethodGet &&
		r.PathValue("group_id") != "" &&
		r.PathValue("subgroup_id") != "" &&
		strings.HasSuffix(r.URL.Path, "/preview"):
		h.previewSubgroupDocument(w, r)

	case r.Method == http.MethodGet &&
		r.PathValue("group_id") != "" &&
		r.PathValue("subgroup_id") != "" &&
		strings.HasSuffix(r.URL.Path, "/download"):
		h.downloadSubgroupDocument(w, r)

	case r.Method == http.MethodGet &&
		strings.HasSuffix(r.URL.Path, "/preview"):
		h.previewPersonalVaultDocument(w, r)

	case r.Method == http.MethodGet &&
		strings.HasSuffix(r.URL.Path, "/preview"):
		h.previewPersonalVaultDocument(w, r)

	case r.Method == http.MethodPost &&
		strings.HasPrefix(r.URL.Path, "/documents/") &&
		strings.HasSuffix(r.URL.Path, "/copy"):
		h.copyPersonalVaultDocument(w, r)

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

	writeJSON(
		w,
		http.StatusOK,
		documents,
	)
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

	path := r.URL.Path

	if !strings.HasPrefix(path, prefix) ||
		!strings.HasSuffix(path, suffix) {
		http.Error(
			w,
			"invalid document path",
			http.StatusBadRequest,
		)
		return
	}

	documentIDText := strings.TrimSuffix(
		strings.TrimPrefix(path, prefix),
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

	filename := strings.NewReplacer(
		"\\",
		"_",
		`"`,
		"_",
		"\r",
		"_",
		"\n",
		"_",
	).Replace(document.OriginalFilename)

	if filename == "" {
		filename = "document"
	}

	w.Header().Set(
		"Content-Type",
		document.MIMEType,
	)

	w.Header().Set(
		"Content-Length",
		strconv.FormatInt(
			document.FileSizeBytes,
			10,
		),
	)

	w.Header().Set(
		"Content-Disposition",
		`inline; filename="`+filename+`"`,
	)

	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, file); err != nil {
		return
	}
}

func (h *Handler) previewSubgroupDocument(
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

	if claims.Role != "college_admin" &&
		claims.Role != "faculty" {
		http.Error(
			w,
			"forbidden",
			http.StatusForbidden,
		)
		return
	}

	groupID, err := strconv.ParseInt(
		r.PathValue("group_id"),
		10,
		64,
	)
	if err != nil || groupID <= 0 {
		http.Error(
			w,
			"invalid group ID",
			http.StatusBadRequest,
		)
		return
	}

	subgroupID, err := strconv.ParseInt(
		r.PathValue("subgroup_id"),
		10,
		64,
	)
	if err != nil || subgroupID <= 0 {
		http.Error(
			w,
			"invalid subgroup ID",
			http.StatusBadRequest,
		)
		return
	}

	const prefix = "/groups/"
	const middle = "/subgroups/"
	const suffix = "/documents/"
	const previewSuffix = "/preview"

	path := r.URL.Path

	if !strings.HasPrefix(path, prefix) ||
		!strings.Contains(path, middle) ||
		!strings.Contains(path, suffix) ||
		!strings.HasSuffix(path, previewSuffix) {
		http.Error(
			w,
			"invalid document path",
			http.StatusBadRequest,
		)
		return
	}

	documentPart := strings.TrimSuffix(
		path,
		previewSuffix,
	)

	documentPart = strings.TrimPrefix(
		documentPart,
		"/groups/"+strconv.FormatInt(groupID, 10)+
			"/subgroups/"+strconv.FormatInt(subgroupID, 10)+
			"/documents/",
	)

	documentID, err := strconv.ParseInt(
		documentPart,
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

	service, ok := h.service.(SubgroupDocumentPreviewService)
	if !ok {
		http.Error(
			w,
			"subgroup document preview service is unavailable",
			http.StatusServiceUnavailable,
		)
		return
	}

	file, document, err := service.PreviewSubgroupDocument(
		r.Context(),
		claims.CollegeID,
		claims.UserID,
		claims.Role,
		groupID,
		subgroupID,
		documentID,
	)
	if err != nil {
		writeDocumentServiceError(w, err)
		return
	}
	defer file.Close()

	filename := strings.NewReplacer(
		"\\",
		"_",
		`"`,
		"_",
		"\r",
		"_",
		"\n",
		"_",
	).Replace(document.OriginalFilename)

	if filename == "" {
		filename = "document"
	}

	w.Header().Set(
		"Content-Type",
		document.MIMEType,
	)

	w.Header().Set(
		"Content-Length",
		strconv.FormatInt(
			document.FileSizeBytes,
			10,
		),
	)

	w.Header().Set(
		"Content-Disposition",
		`inline; filename="`+filename+`"`,
	)

	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, file); err != nil {
		return
	}
}

func (h *Handler) downloadSubgroupDocument(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if claims.Role != "college_admin" && claims.Role != "faculty" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	groupID, err := strconv.ParseInt(r.PathValue("group_id"), 10, 64)
	if err != nil || groupID <= 0 {
		http.Error(w, "invalid group ID", http.StatusBadRequest)
		return
	}

	subgroupID, err := strconv.ParseInt(r.PathValue("subgroup_id"), 10, 64)
	if err != nil || subgroupID <= 0 {
		http.Error(w, "invalid subgroup ID", http.StatusBadRequest)
		return
	}

	documentID, err := strconv.ParseInt(r.PathValue("document_id"), 10, 64)
	if err != nil || documentID <= 0 {
		http.Error(w, "invalid document ID", http.StatusBadRequest)
		return
	}

	service, ok := h.service.(SubgroupDocumentDownloadService)
	if !ok {
		http.Error(w, "document download is unavailable", http.StatusServiceUnavailable)
		return
	}

	file, document, err := service.DownloadSubgroupDocument(
		r.Context(),
		claims.CollegeID,
		claims.UserID,
		claims.Role,
		groupID,
		subgroupID,
		documentID,
	)

	if err != nil {
		writeDocumentServiceError(w, err)
		return
	}

	if file == nil {
		http.Error(
			w,
			"document file is unavailable",
			http.StatusInternalServerError,
		)
		return
	}

	defer file.Close()

	filename := strings.NewReplacer(
		"\\", "_",
		`"`, "_",
		"\r", "_",
		"\n", "_",
	).Replace(document.OriginalFilename)
	if filename == "" {
		filename = "document"
	}

	w.Header().Set("Content-Type", document.MIMEType)
	w.Header().Set("Content-Length", strconv.FormatInt(document.FileSizeBytes, 10))
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)

	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, file)
}

// copyPersonalVaultDocument copies a Personal Vault document into
// a subgroup. The original document remains in the Personal Vault.
func (h *Handler) copyPersonalVaultDocument(
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
	const suffix = "/copy"

	path := r.URL.Path

	if !strings.HasPrefix(path, prefix) ||
		!strings.HasSuffix(path, suffix) {
		http.Error(
			w,
			"invalid document path",
			http.StatusBadRequest,
		)
		return
	}

	documentIDText := strings.TrimSuffix(
		strings.TrimPrefix(path, prefix),
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

	var request struct {
		GroupID    int64 `json:"group_id"`
		SubgroupID int64 `json:"subgroup_id"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	if request.GroupID <= 0 || request.SubgroupID <= 0 {
		http.Error(
			w,
			"invalid group or subgroup ID",
			http.StatusBadRequest,
		)
		return
	}

	// Keep CopyService separate from ServiceInterface so existing
	// service mocks don't have to implement the copy operation.
	copyService, ok := h.service.(CopyService)
	if !ok {
		http.Error(
			w,
			"copy operation unavailable",
			http.StatusInternalServerError,
		)
		return
	}

	document, err := copyService.CopyPersonalVaultDocument(
		r.Context(),
		claims.CollegeID,
		claims.UserID,
		documentID,
		request.GroupID,
		request.SubgroupID,
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

	// Allow multipart overhead while keeping the actual file size
	// limited by ValidateUpload and MaxUploadSize.
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

func (h *Handler) ListSubgroupDocuments(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if claims.Role != "college_admin" && claims.Role != "faculty" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	groupID, err := strconv.ParseInt(r.PathValue("group_id"), 10, 64)
	if err != nil || groupID <= 0 {
		http.Error(w, "invalid group id", http.StatusBadRequest)
		return
	}

	subgroupID, err := strconv.ParseInt(r.PathValue("subgroup_id"), 10, 64)
	if err != nil || subgroupID <= 0 {
		http.Error(w, "invalid subgroup id", http.StatusBadRequest)
		return
	}

	subgroupDocumentService, ok := h.service.(SubgroupDocumentListService)
	if !ok {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}

	documents, err := subgroupDocumentService.ListSubgroupDocuments(
		r.Context(),
		claims.CollegeID,
		claims.UserID,
		claims.Role,
		groupID,
		subgroupID,
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrForbidden):
			http.Error(w, "forbidden", http.StatusForbidden)
		case errors.Is(err, ErrDocumentNotFound):
			http.Error(w, "subgroup not found", http.StatusNotFound)
		case errors.Is(err, ErrInvalidInput):
			http.Error(w, "invalid input", http.StatusBadRequest)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	if err := json.NewEncoder(w).Encode(documents); err != nil {
		return
	}
}
