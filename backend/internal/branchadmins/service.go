package branchadmins

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrInvalidInput     = errors.New("invalid branch admin input")
	ErrBranchNotFound   = errors.New("branch not found")
	ErrUserNotFound     = errors.New("eligible faculty user not found")
	ErrAssignmentExists = errors.New("branch admin assignment already exists")
)

type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Service struct {
	db Querier
}

func NewService(database Querier) *Service {
	return &Service{
		db: database,
	}
}

type Assignment struct {
	ID         int64  `json:"id"`
	CollegeID  int64  `json:"college_id"`
	BranchID   int64  `json:"branch_id"`
	UserID     int64  `json:"user_id"`
	AssignedBy int64  `json:"assigned_by"`
	IsActive   bool   `json:"is_active"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

func (s *Service) Assign(
	ctx context.Context,
	collegeID int64,
	branchID int64,
	userID int64,
	assignedBy int64,
) (Assignment, error) {
	if collegeID <= 0 ||
		branchID <= 0 ||
		userID <= 0 ||
		assignedBy <= 0 {
		return Assignment{}, ErrInvalidInput
	}

	const branchQuery = `
		SELECT id
		FROM branches
		WHERE id = $1
		  AND college_id = $2
		  AND is_active = TRUE
	`

	var foundBranchID int64

	err := s.db.QueryRow(
		ctx,
		branchQuery,
		branchID,
		collegeID,
	).Scan(&foundBranchID)

	if errors.Is(err, pgx.ErrNoRows) {
		return Assignment{}, ErrBranchNotFound
	}

	if err != nil {
		return Assignment{}, fmt.Errorf("check branch: %w", err)
	}

	const userQuery = `
		SELECT id
		FROM users
		WHERE id = $1
		  AND college_id = $2
		  AND is_active = TRUE
		  AND role = 'faculty'
	`

	var foundUserID int64

	err = s.db.QueryRow(
		ctx,
		userQuery,
		userID,
		collegeID,
	).Scan(&foundUserID)

	if errors.Is(err, pgx.ErrNoRows) {
		return Assignment{}, ErrUserNotFound
	}

	if err != nil {
		return Assignment{}, fmt.Errorf("check faculty user: %w", err)
	}

	const assignerQuery = `
		SELECT id
		FROM users
		WHERE id = $1
		  AND college_id = $2
		  AND is_active = TRUE
		  AND role = 'college_admin'
	`

	var foundAssignerID int64

	err = s.db.QueryRow(
		ctx,
		assignerQuery,
		assignedBy,
		collegeID,
	).Scan(&foundAssignerID)

	if errors.Is(err, pgx.ErrNoRows) {
		return Assignment{}, ErrInvalidInput
	}

	if err != nil {
		return Assignment{}, fmt.Errorf("check assigning user: %w", err)
	}

	const insertQuery = `
		INSERT INTO branch_admin_assignments (
			college_id,
			branch_id,
			user_id,
			assigned_by
		)
		VALUES ($1, $2, $3, $4)
		RETURNING
			id,
			college_id,
			branch_id,
			user_id,
			assigned_by,
			is_active,
			created_at::text,
			updated_at::text
	`

	var assignment Assignment

	err = s.db.QueryRow(
		ctx,
		insertQuery,
		collegeID,
		branchID,
		userID,
		assignedBy,
	).Scan(
		&assignment.ID,
		&assignment.CollegeID,
		&assignment.BranchID,
		&assignment.UserID,
		&assignment.AssignedBy,
		&assignment.IsActive,
		&assignment.CreatedAt,
		&assignment.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) &&
			pgErr.Code == "23505" &&
			pgErr.ConstraintName == "uq_branch_admin_assignments_branch_user" {
			return Assignment{}, ErrAssignmentExists
		}

		return Assignment{}, fmt.Errorf("create branch admin assignment: %w", err)
	}

	return assignment, nil
}
