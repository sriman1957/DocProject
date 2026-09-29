package groups

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrForbidden        = errors.New("forbidden")
	ErrInvalidInput     = errors.New("invalid group input")
	ErrGroupNotFound    = errors.New("group not found")
	ErrMemberNotFound   = errors.New("group member not found")
	ErrMembershipExists = errors.New("membership already exists")
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

type AddMemberInput struct {
	UserID int64
}

type Member struct {
	UserID         int64  `json:"user_id"`
	FullName       string `json:"full_name"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	MembershipRole string `json:"membership_role"`
	JoinedAt       string `json:"joined_at"`
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

// ListMembers returns the members of an active group belonging
// to the specified college.
func (s *Service) ListMembers(
	ctx context.Context,
	collegeID int64,
	groupID int64,
) ([]Member, error) {
	if collegeID <= 0 || groupID <= 0 {
		return nil, ErrInvalidInput
	}

	// Verify that the group exists, is active, and belongs
	// to the authenticated user's college.
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
		return nil, fmt.Errorf("list group members: check group: %w", err)
	}

	const membersQuery = `
		SELECT
			u.id,
			u.full_name,
			u.email,
			u.role,
			gm.membership_role,
			gm.joined_at::text
		FROM group_memberships gm
		JOIN users u
		  ON u.id = gm.user_id
		 AND u.college_id = gm.college_id
		WHERE gm.group_id = $1
		  AND gm.college_id = $2
		ORDER BY gm.joined_at ASC, u.id ASC
	`

	rows, err := s.db.Query(
		ctx,
		membersQuery,
		groupID,
		collegeID,
	)
	if err != nil {
		return nil, fmt.Errorf("list group members: query: %w", err)
	}
	defer rows.Close()

	members := make([]Member, 0)

	for rows.Next() {
		var member Member

		if err := rows.Scan(
			&member.UserID,
			&member.FullName,
			&member.Email,
			&member.Role,
			&member.MembershipRole,
			&member.JoinedAt,
		); err != nil {
			return nil, fmt.Errorf("list group members: scan: %w", err)
		}

		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list group members: rows: %w", err)
	}

	return members, nil
}

func (s *Service) AddMember(
	ctx context.Context,
	collegeID int64,
	groupID int64,
	input AddMemberInput,
) (Member, error) {
	if collegeID <= 0 || groupID <= 0 || input.UserID <= 0 {
		return Member{}, ErrInvalidInput
	}

	// Verify that the group belongs to this college and is active.
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
		return Member{}, ErrGroupNotFound
	}

	if err != nil {
		return Member{}, fmt.Errorf("check group: %w", err)
	}

	// Only active students and faculty from the same college can join.
	const userQuery = `
		SELECT
			id,
			full_name,
			email,
			role
		FROM users
		WHERE id = $1
		  AND college_id = $2
		  AND is_active = TRUE
		  AND role IN ('student', 'faculty')
	`

	var member Member

	err = s.db.QueryRow(
		ctx,
		userQuery,
		input.UserID,
		collegeID,
	).Scan(
		&member.UserID,
		&member.FullName,
		&member.Email,
		&member.Role,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return Member{}, ErrMemberNotFound
	}

	if err != nil {
		return Member{}, fmt.Errorf("check user: %w", err)
	}

	// Insert the membership and return the user's public details.
	// The membership role is derived from the user's database role.
	const insertQuery = `
		WITH inserted AS (
			INSERT INTO group_memberships (
				college_id,
				group_id,
				user_id,
				membership_role
			)
			VALUES ($1, $2, $3, $4)
			RETURNING
				college_id,
				user_id,
				membership_role,
				joined_at
		)
		SELECT
			u.id,
			u.full_name,
			u.email,
			u.role,
			i.membership_role,
			i.joined_at::text
		FROM inserted i
		JOIN users u
		  ON u.id = i.user_id
		 AND u.college_id = i.college_id
	`

	err = s.db.QueryRow(
		ctx,
		insertQuery,
		collegeID,
		groupID,
		member.UserID,
		member.Role,
	).Scan(
		&member.UserID,
		&member.FullName,
		&member.Email,
		&member.Role,
		&member.MembershipRole,
		&member.JoinedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) &&
			pgErr.Code == "23505" &&
			pgErr.ConstraintName == "group_memberships_group_id_user_id_key" {
			return Member{}, ErrMembershipExists
		}

		return Member{}, fmt.Errorf("insert group membership: %w", err)
	}

	return member, nil
}
