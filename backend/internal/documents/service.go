package documents

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"docproject/backend/internal/accessperiods"
)

var (
	ErrInvalidInput     = errors.New("invalid document input")
	ErrForbidden        = errors.New("forbidden")
	ErrDocumentNotFound = errors.New("document not found")
)

type Querier interface {
	Query(
		ctx context.Context,
		sql string,
		args ...any,
	) (pgx.Rows, error)

	QueryRow(
		ctx context.Context,
		sql string,
		args ...any,
	) pgx.Row
}

type SubgroupAccessAuthorizer interface {
	AuthorizeStudentAccess(
		ctx context.Context,
		collegeID int64,
		studentID int64,
		groupID int64,
		subgroupID int64,
		at string,
	) error
}

type Service struct {
	db         Querier
	storage    Storage
	authorizer SubgroupAccessAuthorizer
}

func NewService(database Querier) *Service {
	return &Service{
		db: database,
	}
}

func NewServiceWithStorage(
	database Querier,
	storage Storage,
) *Service {
	return &Service{
		db:      database,
		storage: storage,
	}
}

func NewServiceWithStorageAndAuthorizer(
	database Querier,
	storage Storage,
	authorizer SubgroupAccessAuthorizer,
) *Service {
	return &Service{
		db:         database,
		storage:    storage,
		authorizer: authorizer,
	}
}

type Document struct {
	ID               int64     `json:"id"`
	OriginalFilename string    `json:"original_filename"`
	MIMEType         string    `json:"mime_type"`
	FileSizeBytes    int64     `json:"file_size_bytes"`
	SHA256           string    `json:"sha256"`
	UploadedAt       time.Time `json:"uploaded_at"`
}

type UploadedDocument struct {
	ID               int64     `json:"id"`
	OriginalFilename string    `json:"original_filename"`
	MIMEType         string    `json:"mime_type"`
	FileSizeBytes    int64     `json:"file_size_bytes"`
	SHA256           string    `json:"sha256"`
	UploadedAt       time.Time `json:"uploaded_at"`
}

func (s *Service) ListPersonalVault(
	ctx context.Context,
	collegeID int64,
	studentID int64,
) ([]Document, error) {
	if collegeID <= 0 || studentID <= 0 {
		return nil, ErrInvalidInput
	}

	const query = `
		SELECT
			id,
			original_filename,
			mime_type,
			file_size_bytes,
			sha256,
			uploaded_at
		FROM documents
		WHERE college_id = $1
		  AND owner_id = $2
		  AND subgroup_id IS NULL
		  AND deleted_at IS NULL
		ORDER BY uploaded_at DESC, id DESC
	`

	rows, err := s.db.Query(
		ctx,
		query,
		collegeID,
		studentID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"list personal vault documents: query documents: %w",
			err,
		)
	}
	defer rows.Close()

	documents := make([]Document, 0)

	for rows.Next() {
		var document Document

		if err := rows.Scan(
			&document.ID,
			&document.OriginalFilename,
			&document.MIMEType,
			&document.FileSizeBytes,
			&document.SHA256,
			&document.UploadedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"list personal vault documents: scan document: %w",
				err,
			)
		}

		documents = append(documents, document)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"list personal vault documents: iterate documents: %w",
			err,
		)
	}

	return documents, nil
}

// ListSubgroupDocuments returns active documents stored in a subgroup.
//
// College admins can access any subgroup in their college.
// Faculty can access subgroups belonging to groups where they are members.
func (s *Service) ListSubgroupDocuments(
	ctx context.Context,
	collegeID int64,
	actorID int64,
	actorRole string,
	groupID int64,
	subgroupID int64,
) ([]Document, error) {
	if collegeID <= 0 ||
		actorID <= 0 ||
		groupID <= 0 ||
		subgroupID <= 0 {
		return nil, ErrInvalidInput
	}

	if actorRole != "college_admin" && actorRole != "faculty" {
		return nil, ErrForbidden
	}

	// Faculty must belong to the requested group.
	if actorRole == "faculty" {
		const membershipQuery = `
			SELECT EXISTS (
				SELECT 1
				FROM group_memberships
				WHERE college_id = $1
				  AND group_id = $2
				  AND user_id = $3
				  AND membership_role = 'faculty'
			)
		`

		var isMember bool
		err := s.db.QueryRow(
			ctx,
			membershipQuery,
			collegeID,
			groupID,
			actorID,
		).Scan(&isMember)
		if err != nil {
			return nil, fmt.Errorf(
				"list subgroup documents: check faculty membership: %w",
				err,
			)
		}

		if !isMember {
			return nil, ErrForbidden
		}
	}

	// Verify that the subgroup belongs to the requested group and college
	// and is still active.
	const subgroupQuery = `
		SELECT EXISTS (
			SELECT 1
			FROM subgroups
			WHERE id = $1
			  AND group_id = $2
			  AND college_id = $3
			  AND is_active = TRUE
		)
	`

	var subgroupExists bool
	err := s.db.QueryRow(
		ctx,
		subgroupQuery,
		subgroupID,
		groupID,
		collegeID,
	).Scan(&subgroupExists)
	if err != nil {
		return nil, fmt.Errorf(
			"list subgroup documents: check subgroup: %w",
			err,
		)
	}

	if !subgroupExists {
		return nil, ErrDocumentNotFound
	}

	// Return only active documents stored in this subgroup.
	const documentsQuery = `
		SELECT
			id,
			original_filename,
			mime_type,
			file_size_bytes,
			sha256,
			uploaded_at
		FROM documents
		WHERE college_id = $1
		  AND subgroup_id = $2
		  AND deleted_at IS NULL
		ORDER BY uploaded_at DESC, id DESC
	`

	rows, err := s.db.Query(
		ctx,
		documentsQuery,
		collegeID,
		subgroupID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"list subgroup documents: query documents: %w",
			err,
		)
	}
	defer rows.Close()

	documents := make([]Document, 0)

	for rows.Next() {
		var document Document

		if err := rows.Scan(
			&document.ID,
			&document.OriginalFilename,
			&document.MIMEType,
			&document.FileSizeBytes,
			&document.SHA256,
			&document.UploadedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"list subgroup documents: scan document: %w",
				err,
			)
		}

		documents = append(documents, document)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"list subgroup documents: iterate documents: %w",
			err,
		)
	}

	return documents, nil
}

// PreviewSubgroupDocument opens an active document stored in a subgroup.
//
// College admins can access any subgroup in their college.
// Faculty can access subgroups belonging to groups where they are members.
// Students are not allowed to preview subgroup documents.
func (s *Service) PreviewSubgroupDocument(
	ctx context.Context,
	collegeID int64,
	actorID int64,
	actorRole string,
	groupID int64,
	subgroupID int64,
	documentID int64,
) (io.ReadCloser, Document, error) {
	if collegeID <= 0 ||
		actorID <= 0 ||
		groupID <= 0 ||
		subgroupID <= 0 ||
		documentID <= 0 {
		return nil, Document{}, ErrInvalidInput
	}

	if actorRole != "college_admin" && actorRole != "faculty" {
		return nil, Document{}, ErrForbidden
	}

	if s.storage == nil {
		return nil, Document{}, errors.New(
			"document storage is not configured",
		)
	}

	// Faculty must belong to the requested parent group.
	if actorRole == "faculty" {
		const membershipQuery = `
			SELECT EXISTS (
				SELECT 1
				FROM group_memberships
				WHERE college_id = $1
				  AND group_id = $2
				  AND user_id = $3
				  AND membership_role = 'faculty'
			)
		`

		var isMember bool

		err := s.db.QueryRow(
			ctx,
			membershipQuery,
			collegeID,
			groupID,
			actorID,
		).Scan(&isMember)

		if err != nil {
			return nil, Document{}, fmt.Errorf(
				"preview subgroup document: check faculty membership: %w",
				err,
			)
		}

		if !isMember {
			return nil, Document{}, ErrForbidden
		}
	}

	// Verify that the subgroup belongs to the requested group and
	// college and is still active.
	const subgroupQuery = `
		SELECT EXISTS (
			SELECT 1
			FROM subgroups
			WHERE id = $1
			  AND group_id = $2
			  AND college_id = $3
			  AND is_active = TRUE
		)
	`

	var subgroupExists bool

	err := s.db.QueryRow(
		ctx,
		subgroupQuery,
		subgroupID,
		groupID,
		collegeID,
	).Scan(&subgroupExists)

	if err != nil {
		return nil, Document{}, fmt.Errorf(
			"preview subgroup document: check subgroup: %w",
			err,
		)
	}

	if !subgroupExists {
		return nil, Document{}, ErrDocumentNotFound
	}

	// The document must belong to the requested subgroup and
	// must not have been soft-deleted.
	const documentQuery = `
		SELECT
			id,
			original_filename,
			mime_type,
			file_size_bytes,
			sha256,
			uploaded_at,
			storage_key
		FROM documents
		WHERE id = $1
		  AND college_id = $2
		  AND subgroup_id = $3
		  AND deleted_at IS NULL
	`

	var document Document
	var storageKey string

	err = s.db.QueryRow(
		ctx,
		documentQuery,
		documentID,
		collegeID,
		subgroupID,
	).Scan(
		&document.ID,
		&document.OriginalFilename,
		&document.MIMEType,
		&document.FileSizeBytes,
		&document.SHA256,
		&document.UploadedAt,
		&storageKey,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, Document{}, ErrDocumentNotFound
		}

		return nil, Document{}, fmt.Errorf(
			"preview subgroup document: query document: %w",
			err,
		)
	}

	file, err := s.storage.Open(storageKey)
	if err != nil {
		if errors.Is(err, ErrStorageNotFound) {
			return nil, Document{}, ErrDocumentNotFound
		}

		return nil, Document{}, fmt.Errorf(
			"preview subgroup document: open storage object: %w",
			err,
		)
	}

	return file, document, nil
}

func (s *Service) GetPersonalVaultDocument(
	ctx context.Context,
	collegeID int64,
	studentID int64,
	documentID int64,
) (Document, error) {
	if collegeID <= 0 ||
		studentID <= 0 ||
		documentID <= 0 {
		return Document{}, ErrInvalidInput
	}

	const query = `
		SELECT
			id,
			original_filename,
			mime_type,
			file_size_bytes,
			sha256,
			uploaded_at
		FROM documents
		WHERE id = $1
		  AND college_id = $2
		  AND owner_id = $3
		  AND subgroup_id IS NULL
		  AND deleted_at IS NULL
	`

	var document Document

	err := s.db.QueryRow(
		ctx,
		query,
		documentID,
		collegeID,
		studentID,
	).Scan(
		&document.ID,
		&document.OriginalFilename,
		&document.MIMEType,
		&document.FileSizeBytes,
		&document.SHA256,
		&document.UploadedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Document{}, ErrDocumentNotFound
		}

		return Document{}, fmt.Errorf(
			"get personal vault document: query document: %w",
			err,
		)
	}

	return document, nil
}

func (s *Service) PreviewPersonalVaultDocument(
	ctx context.Context,
	collegeID int64,
	studentID int64,
	documentID int64,
) (io.ReadCloser, Document, error) {
	if collegeID <= 0 ||
		studentID <= 0 ||
		documentID <= 0 {
		return nil, Document{}, ErrInvalidInput
	}

	if s.storage == nil {
		return nil, Document{}, errors.New(
			"document storage is not configured",
		)
	}

	const query = `
		SELECT
			id,
			original_filename,
			mime_type,
			file_size_bytes,
			sha256,
			uploaded_at,
			storage_key
		FROM documents
		WHERE id = $1
		  AND college_id = $2
		  AND owner_id = $3
		  AND subgroup_id IS NULL
		  AND deleted_at IS NULL
	`

	var document Document
	var storageKey string

	err := s.db.QueryRow(
		ctx,
		query,
		documentID,
		collegeID,
		studentID,
	).Scan(
		&document.ID,
		&document.OriginalFilename,
		&document.MIMEType,
		&document.FileSizeBytes,
		&document.SHA256,
		&document.UploadedAt,
		&storageKey,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, Document{}, ErrDocumentNotFound
		}

		return nil, Document{}, fmt.Errorf(
			"preview personal vault document: query document: %w",
			err,
		)
	}

	file, err := s.storage.Open(storageKey)
	if err != nil {
		if errors.Is(err, ErrStorageNotFound) {
			return nil, Document{}, ErrDocumentNotFound
		}

		return nil, Document{}, fmt.Errorf(
			"preview personal vault document: open storage object: %w",
			err,
		)
	}

	return file, document, nil
}

func (s *Service) CreatePersonalVaultDocument(
	ctx context.Context,
	collegeID int64,
	studentID int64,
	originalFilename string,
	data []byte,
) (UploadedDocument, error) {
	if collegeID <= 0 || studentID <= 0 {
		return UploadedDocument{}, ErrInvalidInput
	}

	if s.storage == nil {
		return UploadedDocument{}, errors.New(
			"document storage is not configured",
		)
	}

	if strings.TrimSpace(originalFilename) == "" {
		return UploadedDocument{}, ErrInvalidInput
	}

	upload, err := ValidateUpload(data)
	if err != nil {
		return UploadedDocument{}, err
	}

	sha256Hash := CalculateSHA256(data)
	storageKey := GenerateStorageKey()

	if err := s.storage.Save(
		storageKey,
		data,
	); err != nil {
		return UploadedDocument{}, fmt.Errorf(
			"save document: %w",
			err,
		)
	}

	document, err := s.insertDocument(
		ctx,
		collegeID,
		studentID,
		originalFilename,
		storageKey,
		upload.MIMEType,
		upload.Size,
		sha256Hash,
	)
	if err != nil {
		if deleteErr := s.storage.Delete(
			storageKey,
		); deleteErr != nil {
			return UploadedDocument{}, fmt.Errorf(
				"insert document: %w; cleanup stored file: %v",
				err,
				deleteErr,
			)
		}

		return UploadedDocument{}, err
	}

	return document, nil
}

// CopyPersonalVaultDocument creates a separate physical copy of a
// student's Personal Vault document in an authorized subgroup.
// The original document remains unchanged.
func (s *Service) CopyPersonalVaultDocument(
	ctx context.Context,
	collegeID int64,
	studentID int64,
	documentID int64,
	groupID int64,
	subgroupID int64,
) (UploadedDocument, error) {
	if collegeID <= 0 ||
		studentID <= 0 ||
		documentID <= 0 ||
		groupID <= 0 ||
		subgroupID <= 0 {
		return UploadedDocument{}, ErrInvalidInput
	}

	if s.storage == nil {
		return UploadedDocument{}, errors.New(
			"document storage is not configured",
		)
	}

	if s.authorizer == nil {
		return UploadedDocument{}, errors.New(
			"subgroup access authorizer is not configured",
		)
	}

	// Confirm that the source belongs to this student and is an
	// active Personal Vault document, not an existing subgroup copy.
	const sourceQuery = `
		SELECT
			original_filename,
			storage_key,
			mime_type,
			file_size_bytes,
			sha256
		FROM documents
		WHERE id = $1
		  AND college_id = $2
		  AND owner_id = $3
		  AND subgroup_id IS NULL
		  AND deleted_at IS NULL
	`

	var (
		filename     string
		sourceKey    string
		sourceMIME   string
		sourceSHA256 string
		sourceSize   int64
	)

	err := s.db.QueryRow(
		ctx,
		sourceQuery,
		documentID,
		collegeID,
		studentID,
	).Scan(
		&filename,
		&sourceKey,
		&sourceMIME,
		&sourceSize,
		&sourceSHA256,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UploadedDocument{}, ErrDocumentNotFound
		}

		return UploadedDocument{}, fmt.Errorf(
			"copy personal vault document: load source: %w",
			err,
		)
	}

	// Verify group membership, subgroup status, and the active
	// access period before copying any file.
	at := time.Now().UTC().Format(time.RFC3339Nano)

	err = s.authorizer.AuthorizeStudentAccess(
		ctx,
		collegeID,
		studentID,
		groupID,
		subgroupID,
		at,
	)
	if err != nil {
		if errors.Is(err, accessperiods.ErrForbidden) ||
			errors.Is(err, accessperiods.ErrSubgroupNotFound) {
			return UploadedDocument{}, ErrForbidden
		}

		if errors.Is(err, accessperiods.ErrInvalidInput) {
			return UploadedDocument{}, ErrInvalidInput
		}

		return UploadedDocument{}, fmt.Errorf(
			"copy personal vault document: authorize subgroup access: %w",
			err,
		)
	}

	// Read the original file without moving or modifying it.
	sourceFile, err := s.storage.Open(sourceKey)
	if err != nil {
		if errors.Is(err, ErrStorageNotFound) {
			return UploadedDocument{}, ErrDocumentNotFound
		}

		return UploadedDocument{}, fmt.Errorf(
			"copy personal vault document: open source: %w",
			err,
		)
	}

	data, readErr := io.ReadAll(
		io.LimitReader(sourceFile, MaxUploadSize+1),
	)
	closeErr := sourceFile.Close()

	if readErr != nil {
		return UploadedDocument{}, fmt.Errorf(
			"copy personal vault document: read source: %w",
			readErr,
		)
	}

	if closeErr != nil {
		return UploadedDocument{}, fmt.Errorf(
			"copy personal vault document: close source: %w",
			closeErr,
		)
	}

	if int64(len(data)) > MaxUploadSize {
		return UploadedDocument{}, ErrFileTooLarge
	}

	// Revalidate the file and verify its stored metadata before
	// creating a second document record.
	upload, err := ValidateUpload(data)
	if err != nil {
		return UploadedDocument{}, err
	}

	sha256Hash := CalculateSHA256(data)

	if upload.Size != sourceSize ||
		upload.MIMEType != sourceMIME ||
		sha256Hash != strings.TrimSpace(sourceSHA256) {
		return UploadedDocument{}, errors.New(
			"copy personal vault document: source metadata mismatch",
		)
	}

	// Give the copied file its own unique storage key.
	newStorageKey := GenerateStorageKey()

	if err := s.storage.Save(
		newStorageKey,
		data,
	); err != nil {
		return UploadedDocument{}, fmt.Errorf(
			"copy personal vault document: save copy: %w",
			err,
		)
	}

	// Insert the subgroup document and audit record in a single
	// PostgreSQL statement. If either database insert fails,
	// the statement rolls back both inserts.
	const insertCopyQuery = `
		WITH new_document AS (
			INSERT INTO documents (
				college_id,
				owner_id,
				subgroup_id,
				original_filename,
				storage_key,
				mime_type,
				file_size_bytes,
				sha256,
				copied_from_document_id
			)
			VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9
			)
			RETURNING
				id,
				original_filename,
				mime_type,
				file_size_bytes,
				sha256,
				uploaded_at
		),
		new_audit AS (
			INSERT INTO document_audit_logs (
				document_id,
				actor_user_id,
				action,
				filename
			)
			SELECT
				id,
				$2,
				'document_copied',
				original_filename
			FROM new_document
			RETURNING document_id
		)
		SELECT
			d.id,
			d.original_filename,
			d.mime_type,
			d.file_size_bytes,
			d.sha256,
			d.uploaded_at
		FROM new_document d
		JOIN new_audit a ON a.document_id = d.id
	`

	var copied UploadedDocument

	err = s.db.QueryRow(
		ctx,
		insertCopyQuery,
		collegeID,
		studentID,
		subgroupID,
		filename,
		newStorageKey,
		upload.MIMEType,
		upload.Size,
		sha256Hash,
		documentID,
	).Scan(
		&copied.ID,
		&copied.OriginalFilename,
		&copied.MIMEType,
		&copied.FileSizeBytes,
		&copied.SHA256,
		&copied.UploadedAt,
	)
	if err != nil {
		// The database statement is atomic. Remove the physical copy
		// if the database insert or audit insertion fails.
		if deleteErr := s.storage.Delete(newStorageKey); deleteErr != nil &&
			!errors.Is(deleteErr, ErrStorageNotFound) {
			return UploadedDocument{}, fmt.Errorf(
				"insert copied document and audit event: %w; cleanup copied file: %v",
				err,
				deleteErr,
			)
		}

		return UploadedDocument{}, fmt.Errorf(
			"insert copied document and audit event: %w",
			err,
		)
	}

	return copied, nil
}

func (s *Service) insertDocument(
	ctx context.Context,
	collegeID int64,
	studentID int64,
	originalFilename string,
	storageKey string,
	mimeType string,
	fileSize int64,
	sha256Hash string,
) (UploadedDocument, error) {
	const query = `
		INSERT INTO documents (
			college_id,
			owner_id,
			subgroup_id,
			original_filename,
			storage_key,
			mime_type,
			file_size_bytes,
			sha256
		)
		VALUES (
			$1,
			$2,
			NULL,
			$3,
			$4,
			$5,
			$6,
			$7
		)
		RETURNING
			id,
			original_filename,
			mime_type,
			file_size_bytes,
			sha256,
			uploaded_at
	`

	var document UploadedDocument

	err := s.db.QueryRow(
		ctx,
		query,
		collegeID,
		studentID,
		originalFilename,
		storageKey,
		mimeType,
		fileSize,
		sha256Hash,
	).Scan(
		&document.ID,
		&document.OriginalFilename,
		&document.MIMEType,
		&document.FileSizeBytes,
		&document.SHA256,
		&document.UploadedAt,
	)
	if err != nil {
		return UploadedDocument{}, fmt.Errorf(
			"insert document: %w",
			err,
		)
	}

	return document, nil
}
