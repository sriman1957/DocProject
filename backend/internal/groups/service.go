package groups

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

var (
	ErrForbidden     = errors.New("forbidden")
	ErrInvalidInput  = errors.New("invalid group input")
	ErrGroupNotFound = errors.New("group not found")
)

type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type Service struct {
	db Querier
}

func NewService(database Querier) *Service {
	return &Service{
		db: database,
	}
}

type Group struct {
	ID          int64  `json:"id"`
	CollegeID   int64  `json:"college_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedBy   int64  `json:"created_by"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type CreateInput struct {
	Name        string
	Description string
}

func (s *Service) Create(
	ctx context.Context,
	collegeID int64,
	createdBy int64,
	input CreateInput,
) (Group, error) {
	name := strings.TrimSpace(input.Name)
	description := strings.TrimSpace(input.Description)

	if collegeID <= 0 || createdBy <= 0 {
		return Group{}, ErrInvalidInput
	}

	if name == "" || len(name) > 200 {
		return Group{}, ErrInvalidInput
	}

	if len(description) > 10000 {
		return Group{}, ErrInvalidInput
	}

	const query = `
		INSERT INTO groups (
			college_id,
			name,
			description,
			created_by
		)
		VALUES ($1, $2, NULLIF($3, ''), $4)
		RETURNING
			id,
			college_id,
			name,
			COALESCE(description, ''),
			created_by,
			is_active,
			created_at::text,
			updated_at::text
	`

	var group Group

	err := s.db.QueryRow(
		ctx,
		query,
		collegeID,
		name,
		description,
		createdBy,
	).Scan(
		&group.ID,
		&group.CollegeID,
		&group.Name,
		&group.Description,
		&group.CreatedBy,
		&group.IsActive,
		&group.CreatedAt,
		&group.UpdatedAt,
	)
	if err != nil {
		return Group{}, fmt.Errorf("create group: %w", err)
	}

	return group, nil
}

func (s *Service) List(
	ctx context.Context,
	collegeID int64,
) ([]Group, error) {
	if collegeID <= 0 {
		return nil, ErrInvalidInput
	}

	const query = `
		SELECT
			id,
			college_id,
			name,
			COALESCE(description, ''),
			created_by,
			is_active,
			created_at::text,
			updated_at::text
		FROM groups
		WHERE college_id = $1
		  AND is_active = TRUE
		ORDER BY created_at DESC, id DESC
	`

	rows, err := s.db.Query(ctx, query, collegeID)
	if err != nil {
		return nil, fmt.Errorf("list groups: query: %w", err)
	}
	defer rows.Close()

	groups := make([]Group, 0)

	for rows.Next() {
		var group Group

		if err := rows.Scan(
			&group.ID,
			&group.CollegeID,
			&group.Name,
			&group.Description,
			&group.CreatedBy,
			&group.IsActive,
			&group.CreatedAt,
			&group.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("list groups: scan: %w", err)
		}

		groups = append(groups, group)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list groups: rows: %w", err)
	}

	return groups, nil
}
