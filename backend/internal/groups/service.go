package groups

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"docproject/backend/internal/branchadmins"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrForbidden        = errors.New("forbidden")
	ErrInvalidInput     = errors.New("invalid group input")
	ErrGroupNotFound    = errors.New("group not found")
	ErrMemberNotFound   = errors.New("group member not found")
	ErrMembershipExists = errors.New("membership already exists")
	ErrBranchNotFound   = errors.New("branch not found")
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
	BranchID    int64  `json:"branch_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedBy   int64  `json:"created_by"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type CreateInput struct {
	BranchID    int64
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

// authorizeBranch verifies that the caller is allowed to manage
// resources belonging to the specified branch.
//
// College admins have access to every branch in their college.
// Faculty users only have access when they have an active
// Branch Admin assignment for that branch.
//
// Students and all other roles are denied.
func (s *Service) authorizeBranch(
	ctx context.Context,
	collegeID int64,
	userID int64,
	role string,
	branchID int64,
) error {
	if collegeID <= 0 || userID <= 0 || branchID <= 0 {
		return ErrInvalidInput
	}

	switch role {
	case "college_admin":
		return nil

	case "faculty":
		isBranchAdmin, err := branchadmins.IsBranchAdmin(
			ctx,
			s.db,
			collegeID,
			userID,
			branchID,
		)
		if err != nil {
			return fmt.Errorf("check branch admin authorization: %w", err)
		}

		if !isBranchAdmin {
			return ErrForbidden
		}

		return nil

	default:
		return ErrForbidden
	}
}

// Create creates a new batch group inside a branch.
//
// College admins can create a group in any active branch in their college.
// Branch admins can create a group only in branches assigned to them.
func (s *Service) Create(
	ctx context.Context,
	collegeID int64,
	createdBy int64,
	role string,
	input CreateInput,
) (Group, error) {
	name := strings.TrimSpace(input.Name)
	description := strings.TrimSpace(input.Description)

	if collegeID <= 0 || createdBy <= 0 || input.BranchID <= 0 {
		return Group{}, ErrInvalidInput
	}

	if name == "" || len(name) > 200 {
		return Group{}, ErrInvalidInput
	}

	if len(description) > 10000 {
		return Group{}, ErrInvalidInput
	}

	// Verify that the branch belongs to the caller's college
	// and is active.
	const branchQuery = `
		SELECT id
		FROM branches
		WHERE id = $1
		  AND college_id = $2
		  AND is_active = TRUE
	`

	var branchID int64

	err := s.db.QueryRow(
		ctx,
		branchQuery,
		input.BranchID,
		collegeID,
	).Scan(&branchID)

	if errors.Is(err, pgx.ErrNoRows) {
		return Group{}, ErrBranchNotFound
	}

	if err != nil {
		return Group{}, fmt.Errorf("check branch: %w", err)
	}

	// Authorization is checked only after confirming that the branch
	// belongs to the caller's college.
	if err := s.authorizeBranch(
		ctx,
		collegeID,
		createdBy,
		role,
		branchID,
	); err != nil {
		return Group{}, err
	}

	const query = `
		INSERT INTO groups (
			college_id,
			branch_id,
			name,
			description,
			created_by
		)
		VALUES (
			$1,
			$2,
			$3,
			NULLIF($4, ''),
			$5
		)
		RETURNING
			id,
			college_id,
			branch_id,
			name,
			COALESCE(description, ''),
			created_by,
			is_active,
			created_at::text,
			updated_at::text
	`

	var group Group

	err = s.db.QueryRow(
		ctx,
		query,
		collegeID,
		input.BranchID,
		name,
		description,
		createdBy,
	).Scan(
		&group.ID,
		&group.CollegeID,
		&group.BranchID,
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

// List returns all active groups the caller is authorized to see.
//
// College admins can see every active group in their college.
// Branch admins can only see groups belonging to their assigned branches.
func (s *Service) List(
	ctx context.Context,
	collegeID int64,
	userID int64,
	role string,
) ([]Group, error) {
	if collegeID <= 0 || userID <= 0 {
		return nil, ErrInvalidInput
	}

	switch role {
	case "college_admin":
		// No branch restriction.

	case "faculty":
		// Authorization is enforced directly in the query below.

	default:
		return nil, ErrForbidden
	}

	var query string
	var args []any

	if role == "college_admin" {
		query = `
			SELECT
				id,
				college_id,
				branch_id,
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

		args = []any{
			collegeID,
		}
	} else {
		query = `
			SELECT
				g.id,
				g.college_id,
				g.branch_id,
				g.name,
				COALESCE(g.description, ''),
				g.created_by,
				g.is_active,
				g.created_at::text,
				g.updated_at::text
			FROM groups g
			WHERE g.college_id = $1
			  AND g.is_active = TRUE
			  AND EXISTS (
				  SELECT 1
				  FROM branch_admin_assignments baa
				  WHERE baa.college_id = g.college_id
				    AND baa.branch_id = g.branch_id
				    AND baa.user_id = $2
				    AND baa.is_active = TRUE
			  )
			ORDER BY g.created_at DESC, g.id DESC
		`

		args = []any{
			collegeID,
			userID,
		}
	}

	rows, err := s.db.Query(ctx, query, args...)
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
			&group.BranchID,
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

// ListMembers returns the members of an active group.
//
// Access is determined from the group's branch:
//   - college_admin: any branch in their college
//   - faculty: only if they are an active Branch Admin for that branch
func (s *Service) ListMembers(
	ctx context.Context,
	collegeID int64,
	userID int64,
	role string,
	groupID int64,
) ([]Member, error) {
	if collegeID <= 0 || userID <= 0 || groupID <= 0 {
		return nil, ErrInvalidInput
	}

	// Resolve the group's branch while simultaneously enforcing
	// college isolation.
	const groupQuery = `
		SELECT
			id,
			branch_id
		FROM groups
		WHERE id = $1
		  AND college_id = $2
		  AND is_active = TRUE
	`

	var foundGroupID int64
	var branchID int64

	err := s.db.QueryRow(
		ctx,
		groupQuery,
		groupID,
		collegeID,
	).Scan(
		&foundGroupID,
		&branchID,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGroupNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("list group members: check group: %w", err)
	}

	if err := s.authorizeBranch(
		ctx,
		collegeID,
		userID,
		role,
		branchID,
	); err != nil {
		return nil, err
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

// AddMember adds an active student or faculty user to an active group.
//
// The caller must be:
//   - a college admin in the same college, or
//   - a Branch Admin assigned to the group's branch.
func (s *Service) AddMember(
	ctx context.Context,
	collegeID int64,
	userID int64,
	role string,
	groupID int64,
	input AddMemberInput,
) (Member, error) {
	if collegeID <= 0 || userID <= 0 || groupID <= 0 || input.UserID <= 0 {
		return Member{}, ErrInvalidInput
	}

	// Verify that the group belongs to this college, is active,
	// and determine its branch.
	const groupQuery = `
		SELECT
			id,
			branch_id
		FROM groups
		WHERE id = $1
		  AND college_id = $2
		  AND is_active = TRUE
	`

	var foundGroupID int64
	var branchID int64

	err := s.db.QueryRow(
		ctx,
		groupQuery,
		groupID,
		collegeID,
	).Scan(
		&foundGroupID,
		&branchID,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return Member{}, ErrGroupNotFound
	}

	if err != nil {
		return Member{}, fmt.Errorf("check group: %w", err)
	}

	if err := s.authorizeBranch(
		ctx,
		collegeID,
		userID,
		role,
		branchID,
	); err != nil {
		return Member{}, err
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
