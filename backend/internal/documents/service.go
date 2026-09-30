package documents

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidInput = errors.New("invalid document input")
	ErrForbidden    = errors.New("forbidden")
)

type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type Service struct {
	db Querier
}

func NewService(database Querier) *Service {
	return &Service{db: database}
}

type Document struct {
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

	rows, err := s.db.Query(ctx, query, collegeID, studentID)
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
