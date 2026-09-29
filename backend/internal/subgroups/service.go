package subgroups

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

var (
	ErrForbidden      = errors.New("forbidden")
	ErrInvalidInput   = errors.New("invalid subgroup input")
	ErrGroupNotFound  = errors.New("group not found")
	ErrSubgroupExists = errors.New("active subgroup with this name already exists")
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

type Subgroup struct {
	ID          int64  `json:"id"`
	CollegeID   int64  `json:"college_id"`
	GroupID     int64  `json:"group_id"`
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

// Create creates a subgroup in an active group.
//
// College admins can create subgroups in their college.
// Faculty can create subgroups only in groups where they
// have an active faculty membership.
func (s *Service) Create(
	ctx context.Context,
	collegeID int64,
	actorID int64,
	actorRole string,
	groupID int64,
	input CreateInput,
) (Subgroup, error) {
	name := strings.TrimSpace(input.Name)
	description := strings.TrimSpace(input.Description)

	if collegeID <= 0 || actorID <= 0 || groupID <= 0 {
		return Subgroup{}, ErrInvalidInput
	}

	if name == "" || len(name) > 200 {
		return Subgroup{}, ErrInvalidInput
	}

	if len(description) > 10000 {
		return Subgroup{}, ErrInvalidInput
	}

	if actorRole != "college_admin" && actorRole != "faculty" {
		return Subgroup{}, ErrForbidden
	}

	// Verify that the group is active and belongs to this college.
	const groupQuery = `
		SELECT id
		FROM groups
		WHERE id = $1
		  AND college_id = $2
		  AND is_active = TRUE
	`

	var foundGroupID int64

	err := s.db.QueryRow(
		ctx,
		groupQuery,
		groupID,
		collegeID,
	).Scan(&foundGroupID)

	if errors.Is(err, pgx.ErrNoRows) {
		return Subgroup{}, ErrGroupNotFound
	}

	if err != nil {
		return Subgroup{}, fmt.Errorf(
			"create subgroup: check group: %w",
			err,
		)
	}

	// Faculty must have an active faculty membership in this group.
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
			return Subgroup{}, fmt.Errorf(
				"create subgroup: check faculty membership: %w",
				err,
			)
		}

		if !isMember {
			return Subgroup{}, ErrForbidden
		}
	}

	// The partial unique index allows a previously archived subgroup
	// name to be reused, but prevents duplicate active names.
	const insertQuery = `
		INSERT INTO subgroups (
			college_id,
			group_id,
			name,
			description,
			created_by
		)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5)
		ON CONFLICT (group_id, name)
			WHERE is_active = TRUE
		DO NOTHING
		RETURNING
			id,
			college_id,
			group_id,
			name,
			COALESCE(description, ''),
			created_by,
			is_active,
			created_at::text,
			updated_at::text
	`

	var subgroup Subgroup

	err = s.db.QueryRow(
		ctx,
		insertQuery,
		collegeID,
		groupID,
		name,
		description,
		actorID,
	).Scan(
		&subgroup.ID,
		&subgroup.CollegeID,
		&subgroup.GroupID,
		&subgroup.Name,
		&subgroup.Description,
		&subgroup.CreatedBy,
		&subgroup.IsActive,
		&subgroup.CreatedAt,
		&subgroup.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return Subgroup{}, ErrSubgroupExists
	}

	if err != nil {
		return Subgroup{}, fmt.Errorf(
			"create subgroup: insert: %w",
			err,
		)
	}

	return subgroup, nil
}

// List returns active subgroups for a group.
//
// College admins can list subgroups in their college.
// Faculty and students must belong to the requested group.
func (s *Service) List(
	ctx context.Context,
	collegeID int64,
	actorID int64,
	actorRole string,
	groupID int64,
) ([]Subgroup, error) {
	if collegeID <= 0 || actorID <= 0 || groupID <= 0 {
		return nil, ErrInvalidInput
	}

	if actorRole != "college_admin" &&
		actorRole != "faculty" &&
		actorRole != "student" {
		return nil, ErrForbidden
	}

	// Verify that the group is active and belongs to this college.
	const groupQuery = `
		SELECT id
		FROM groups
		WHERE id = $1
		  AND college_id = $2
		  AND is_active = TRUE
	`

	var foundGroupID int64

	err := s.db.QueryRow(
		ctx,
		groupQuery,
		groupID,
		collegeID,
	).Scan(&foundGroupID)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGroupNotFound
	}

	if err != nil {
		return nil, fmt.Errorf(
			"list subgroups: check group: %w",
			err,
		)
	}

	// Non-admin users must be members of the requested group.
	if actorRole != "college_admin" {
		const membershipQuery = `
			SELECT EXISTS (
				SELECT 1
				FROM group_memberships
				WHERE college_id = $1
				  AND group_id = $2
				  AND user_id = $3
				  AND membership_role = $4
			)
		`

		var isMember bool

		err := s.db.QueryRow(
			ctx,
			membershipQuery,
			collegeID,
			groupID,
			actorID,
			actorRole,
		).Scan(&isMember)

		if err != nil {
			return nil, fmt.Errorf(
				"list subgroups: check membership: %w",
				err,
			)
		}

		if !isMember {
			return nil, ErrForbidden
		}
	}

	const listQuery = `
		SELECT
			id,
			college_id,
			group_id,
			name,
			COALESCE(description, ''),
			created_by,
			is_active,
			created_at::text,
			updated_at::text
		FROM subgroups
		WHERE college_id = $1
		  AND group_id = $2
		  AND is_active = TRUE
		ORDER BY created_at ASC, id ASC
	`

	rows, err := s.db.Query(
		ctx,
		listQuery,
		collegeID,
		groupID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"list subgroups: query: %w",
			err,
		)
	}
	defer rows.Close()

	result := make([]Subgroup, 0)

	for rows.Next() {
		var subgroup Subgroup

		if err := rows.Scan(
			&subgroup.ID,
			&subgroup.CollegeID,
			&subgroup.GroupID,
			&subgroup.Name,
			&subgroup.Description,
			&subgroup.CreatedBy,
			&subgroup.IsActive,
			&subgroup.CreatedAt,
			&subgroup.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"list subgroups: scan: %w",
				err,
			)
		}

		result = append(result, subgroup)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"list subgroups: rows: %w",
			err,
		)
	}

	return result, nil
}
