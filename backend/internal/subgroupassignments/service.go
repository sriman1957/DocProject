package subgroupassignments

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidInput       = errors.New("invalid assignment input")
	ErrForbidden          = errors.New("forbidden")
	ErrGroupNotFound      = errors.New("group not found")
	ErrSubgroupNotFound   = errors.New("subgroup not found")
	ErrFacultyNotEligible = errors.New("faculty is not eligible for this subgroup")
	ErrAssignmentExists   = errors.New("active assignment already exists")
	ErrAssignmentNotFound = errors.New("active assignment not found")
)

type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type Service struct {
	db Querier
}

func NewService(database Querier) *Service {
	return &Service{db: database}
}

type Assignment struct {
	ID         int64  `json:"id"`
	CollegeID  int64  `json:"college_id"`
	SubgroupID int64  `json:"subgroup_id"`
	FacultyID  int64  `json:"faculty_id"`
	AssignedBy int64  `json:"assigned_by"`
	AssignedAt string `json:"assigned_at"`
	RevokedBy  *int64 `json:"revoked_by"`
	RevokedAt  string `json:"revoked_at,omitempty"`
}

// checkGroupAccess verifies that the group exists in the actor's college
// and that the actor is authorized to manage its subgroup assignments.
func (s *Service) checkGroupAccess(
	ctx context.Context,
	collegeID int64,
	actorID int64,
	actorRole string,
	groupID int64,
) error {
	if collegeID <= 0 || actorID <= 0 || groupID <= 0 {
		return ErrInvalidInput
	}

	if actorRole != "college_admin" && actorRole != "faculty" {
		return ErrForbidden
	}

	const groupQuery = `
		SELECT EXISTS (
			SELECT 1
			FROM groups
			WHERE id = $1
			  AND college_id = $2
			  AND is_active = TRUE
		)
	`

	var groupExists bool
	err := s.db.QueryRow(
		ctx,
		groupQuery,
		groupID,
		collegeID,
	).Scan(&groupExists)
	if err != nil {
		return fmt.Errorf("check assignment group: %w", err)
	}
	if !groupExists {
		return ErrGroupNotFound
	}

	// College admins can manage assignments throughout their college.
	if actorRole == "college_admin" {
		return nil
	}

	// Faculty must be a faculty member of the parent group.
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
	err = s.db.QueryRow(
		ctx,
		membershipQuery,
		collegeID,
		groupID,
		actorID,
	).Scan(&isMember)
	if err != nil {
		return fmt.Errorf("check assignment actor membership: %w", err)
	}
	if !isMember {
		return ErrForbidden
	}

	return nil
}

// checkSubgroup verifies that the active subgroup belongs to the
// specified group and college.
func (s *Service) checkSubgroup(
	ctx context.Context,
	collegeID int64,
	groupID int64,
	subgroupID int64,
) error {
	if subgroupID <= 0 {
		return ErrInvalidInput
	}

	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM subgroups
			WHERE id = $1
			  AND group_id = $2
			  AND college_id = $3
			  AND is_active = TRUE
		)
	`

	var exists bool
	err := s.db.QueryRow(
		ctx,
		query,
		subgroupID,
		groupID,
		collegeID,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check assignment subgroup: %w", err)
	}
	if !exists {
		return ErrSubgroupNotFound
	}

	return nil
}

// Create assigns an eligible faculty member to an active subgroup.
// College admins and faculty members of the parent group may assign.
func (s *Service) Create(
	ctx context.Context,
	collegeID int64,
	actorID int64,
	actorRole string,
	groupID int64,
	subgroupID int64,
	facultyID int64,
) (Assignment, error) {
	if collegeID <= 0 || actorID <= 0 ||
		groupID <= 0 || subgroupID <= 0 || facultyID <= 0 {
		return Assignment{}, ErrInvalidInput
	}

	if err := s.checkGroupAccess(
		ctx, collegeID, actorID, actorRole, groupID,
	); err != nil {
		return Assignment{}, err
	}

	if err := s.checkSubgroup(
		ctx, collegeID, groupID, subgroupID,
	); err != nil {
		return Assignment{}, err
	}

	// The target must be a faculty user in this college and must
	// have an active faculty membership in the parent group.
	const eligibleQuery = `
		SELECT EXISTS (
			SELECT 1
			FROM users u
			JOIN group_memberships gm
			  ON gm.user_id = u.id
			 AND gm.college_id = u.college_id
			WHERE u.id = $1
			  AND u.college_id = $2
			  AND u.role = 'faculty'
			  AND u.is_active = TRUE
			  AND gm.group_id = $3
			  AND gm.membership_role = 'faculty'
		)
	`

	var eligible bool
	err := s.db.QueryRow(
		ctx,
		eligibleQuery,
		facultyID,
		collegeID,
		groupID,
	).Scan(&eligible)
	if err != nil {
		return Assignment{}, fmt.Errorf(
			"check assignment faculty eligibility: %w", err,
		)
	}
	if !eligible {
		return Assignment{}, ErrFacultyNotEligible
	}

	// ON CONFLICT handles concurrent attempts to create the same
	// active assignment. A revoked assignment does not block reassignment.
	const insertQuery = `
		INSERT INTO subgroup_faculty_assignments (
			college_id,
			subgroup_id,
			faculty_id,
			assigned_by
		)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (subgroup_id, faculty_id)
			WHERE revoked_at IS NULL
		DO NOTHING
		RETURNING
			id,
			college_id,
			subgroup_id,
			faculty_id,
			assigned_by,
			assigned_at::text,
			revoked_by,
			COALESCE(revoked_at::text, '')
	`

	var assignment Assignment
	err = s.db.QueryRow(
		ctx,
		insertQuery,
		collegeID,
		subgroupID,
		facultyID,
		actorID,
	).Scan(
		&assignment.ID,
		&assignment.CollegeID,
		&assignment.SubgroupID,
		&assignment.FacultyID,
		&assignment.AssignedBy,
		&assignment.AssignedAt,
		&assignment.RevokedBy,
		&assignment.RevokedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return Assignment{}, ErrAssignmentExists
	}
	if err != nil {
		return Assignment{}, fmt.Errorf("create subgroup assignment: %w", err)
	}

	return assignment, nil
}

// List returns assignment records for the specified subgroup.
// Revoked records are included to preserve assignment history.
func (s *Service) List(
	ctx context.Context,
	collegeID int64,
	actorID int64,
	actorRole string,
	groupID int64,
	subgroupID int64,
) ([]Assignment, error) {
	if collegeID <= 0 || actorID <= 0 ||
		groupID <= 0 || subgroupID <= 0 {
		return nil, ErrInvalidInput
	}

	if err := s.checkGroupAccess(
		ctx, collegeID, actorID, actorRole, groupID,
	); err != nil {
		return nil, err
	}

	if err := s.checkSubgroup(
		ctx, collegeID, groupID, subgroupID,
	); err != nil {
		return nil, err
	}

	const query = `
		SELECT
			id,
			college_id,
			subgroup_id,
			faculty_id,
			assigned_by,
			assigned_at::text,
			revoked_by,
			COALESCE(revoked_at::text, '')
		FROM subgroup_faculty_assignments
		WHERE college_id = $1
		  AND subgroup_id = $2
		ORDER BY assigned_at ASC, id ASC
	`

	rows, err := s.db.Query(ctx, query, collegeID, subgroupID)
	if err != nil {
		return nil, fmt.Errorf("list subgroup assignments: %w", err)
	}
	defer rows.Close()

	result := make([]Assignment, 0)

	for rows.Next() {
		var assignment Assignment
		if err := rows.Scan(
			&assignment.ID,
			&assignment.CollegeID,
			&assignment.SubgroupID,
			&assignment.FacultyID,
			&assignment.AssignedBy,
			&assignment.AssignedAt,
			&assignment.RevokedBy,
			&assignment.RevokedAt,
		); err != nil {
			return nil, fmt.Errorf("scan subgroup assignment: %w", err)
		}

		result = append(result, assignment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subgroup assignments: %w", err)
	}

	return result, nil
}

// Revoke revokes an active assignment without deleting its history.
func (s *Service) Revoke(
	ctx context.Context,
	collegeID int64,
	actorID int64,
	actorRole string,
	groupID int64,
	subgroupID int64,
	facultyID int64,
) (Assignment, error) {
	if collegeID <= 0 || actorID <= 0 ||
		groupID <= 0 || subgroupID <= 0 || facultyID <= 0 {
		return Assignment{}, ErrInvalidInput
	}

	if err := s.checkGroupAccess(
		ctx, collegeID, actorID, actorRole, groupID,
	); err != nil {
		return Assignment{}, err
	}

	if err := s.checkSubgroup(
		ctx, collegeID, groupID, subgroupID,
	); err != nil {
		return Assignment{}, err
	}

	const query = `
		UPDATE subgroup_faculty_assignments
		SET revoked_by = $1,
		    revoked_at = NOW()
		WHERE college_id = $2
		  AND subgroup_id = $3
		  AND faculty_id = $4
		  AND revoked_at IS NULL
		RETURNING
			id,
			college_id,
			subgroup_id,
			faculty_id,
			assigned_by,
			assigned_at::text,
			revoked_by,
			revoked_at::text
	`

	var assignment Assignment
	err := s.db.QueryRow(
		ctx,
		query,
		actorID,
		collegeID,
		subgroupID,
		facultyID,
	).Scan(
		&assignment.ID,
		&assignment.CollegeID,
		&assignment.SubgroupID,
		&assignment.FacultyID,
		&assignment.AssignedBy,
		&assignment.AssignedAt,
		&assignment.RevokedBy,
		&assignment.RevokedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return Assignment{}, ErrAssignmentNotFound
	}
	if err != nil {
		return Assignment{}, fmt.Errorf("revoke subgroup assignment: %w", err)
	}

	return assignment, nil
}
