package documents

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
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

type Service struct {
	db      Querier
	storage Storage
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
