package accessperiods

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidInput      = errors.New("invalid access period input")
	ErrForbidden         = errors.New("forbidden")
	ErrGroupNotFound     = errors.New("group not found")
	ErrSubgroupNotFound  = errors.New("subgroup not found")
	ErrPeriodOverlap     = errors.New("access period overlaps an existing period")
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

type AccessPeriod struct {
	ID        int64  `json:"id"`
	SubgroupID int64 `json:"subgroup_id"`
	CollegeID int64  `json:"college_id"`
	StartsAt  string `json:"starts_at"`
	EndsAt    string `json:"ends_at"`
	CreatedBy int64  `json:"created_by"`
	CreatedAt string `json:"created_at"`
}

type CreateInput struct {
	StartsAt time.Time
	EndsAt   time.Time
}

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
			SELECT 1 FROM groups
			WHERE id = $1 AND college_id = $2 AND is_active = TRUE
		)
	`

	var exists bool
	if err := s.db.QueryRow(ctx, groupQuery, groupID, collegeID).Scan(&exists); err != nil {
		return fmt.Errorf("check access period group: %w", err)
	}
	if !exists {
		return ErrGroupNotFound
	}

	if actorRole == "college_admin" {
		return nil
	}

	const membershipQuery = `
		SELECT EXISTS (
			SELECT 1 FROM group_memberships
			WHERE college_id = $1
			  AND group_id = $2
			  AND user_id = $3
			  AND membership_role = 'faculty'
		)
	`

	var member bool
	if err := s.db.QueryRow(
		ctx, membershipQuery, collegeID, groupID, actorID,
	).Scan(&member); err != nil {
		return fmt.Errorf("check access period faculty membership: %w", err)
	}
	if !member {
		return ErrForbidden
	}

	return nil
}

func (s *Service) checkSubgroup(
	ctx context.Context,
	collegeID int64,
	groupID int64,
	subgroupID int64,
) error {
	if collegeID <= 0 || groupID <= 0 || subgroupID <= 0 {
		return ErrInvalidInput
	}

	const query = `
		SELECT EXISTS (
			SELECT 1 FROM subgroups
			WHERE id = $1
			  AND group_id = $2
			  AND college_id = $3
			  AND is_active = TRUE
		)
	`

	var exists bool
	if err := s.db.QueryRow(
		ctx, query, subgroupID, groupID, collegeID,
	).Scan(&exists); err != nil {
		return fmt.Errorf("check access period subgroup: %w", err)
	}
	if !exists {
		return ErrSubgroupNotFound
	}

	return nil
}

func (s *Service) checkFacultyAssignment(
	ctx context.Context,
	collegeID int64,
	subgroupID int64,
	actorID int64,
) error {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM subgroup_faculty_assignments
			WHERE college_id = $1
			  AND subgroup_id = $2
			  AND faculty_id = $3
			  AND revoked_at IS NULL
		)
	`

	var assigned bool
	if err := s.db.QueryRow(
		ctx, query, collegeID, subgroupID, actorID,
	).Scan(&assigned); err != nil {
		return fmt.Errorf("check subgroup faculty assignment: %w", err)
	}
	if !assigned {
		return ErrForbidden
	}

	return nil
}

func (s *Service) Create(
	ctx context.Context,
	collegeID int64,
	actorID int64,
	actorRole string,
	groupID int64,
	subgroupID int64,
	input CreateInput,
) (AccessPeriod, error) {
	if input.StartsAt.IsZero() || input.EndsAt.IsZero() ||
		!input.EndsAt.After(input.StartsAt) {
		return AccessPeriod{}, ErrInvalidInput
	}

	if err := s.checkGroupAccess(
		ctx, collegeID, actorID, actorRole, groupID,
	); err != nil {
		return AccessPeriod{}, err
	}

	if err := s.checkSubgroup(
		ctx, collegeID, groupID, subgroupID,
	); err != nil {
		return AccessPeriod{}, err
	}

	if actorRole == "faculty" {
		if err := s.checkFacultyAssignment(
			ctx, collegeID, subgroupID, actorID,
		); err != nil {
			return AccessPeriod{}, err
		}
	}

	const query = `
		INSERT INTO subgroup_access_periods (
			subgroup_id,
			college_id,
			starts_at,
			ends_at,
			created_by
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING
			id,
			subgroup_id,
			college_id,
			starts_at::text,
			ends_at::text,
			created_by,
			created_at::text
	`

	var period AccessPeriod
	err := s.db.QueryRow(
		ctx,
		query,
		subgroupID,
		collegeID,
		input.StartsAt,
		input.EndsAt,
		actorID,
	).Scan(
		&period.ID,
		&period.SubgroupID,
		&period.CollegeID,
		&period.StartsAt,
		&period.EndsAt,
		&period.CreatedBy,
		&period.CreatedAt,
	)
	if err != nil {
		if isExclusionViolation(err) {
			return AccessPeriod{}, ErrPeriodOverlap
		}
		return AccessPeriod{}, fmt.Errorf("create access period: %w", err)
	}

	return period, nil
}

func (s *Service) List(
	ctx context.Context,
	collegeID int64,
	actorID int64,
	actorRole string,
	groupID int64,
	subgroupID int64,
) ([]AccessPeriod, error) {
	if collegeID <= 0 || actorID <= 0 || groupID <= 0 || subgroupID <= 0 {
		return nil, ErrInvalidInput
	}

	if actorRole != "college_admin" &&
		actorRole != "faculty" &&
		actorRole != "student" {
		return nil, ErrForbidden
	}

	if err := s.checkGroupAccessForList(
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
			subgroup_id,
			college_id,
			starts_at::text,
			ends_at::text,
			created_by,
			created_at::text
		FROM subgroup_access_periods
		WHERE college_id = $1
		  AND subgroup_id = $2
		ORDER BY starts_at ASC, id ASC
	`

	rows, err := s.db.Query(ctx, query, collegeID, subgroupID)
	if err != nil {
		return nil, fmt.Errorf("list access periods: %w", err)
	}
	defer rows.Close()

	result := make([]AccessPeriod, 0)
	for rows.Next() {
		var period AccessPeriod
		if err := rows.Scan(
			&period.ID,
			&period.SubgroupID,
			&period.CollegeID,
			&period.StartsAt,
			&period.EndsAt,
			&period.CreatedBy,
			&period.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan access period: %w", err)
		}
		result = append(result, period)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate access periods: %w", err)
	}

	return result, nil
}

func (s *Service) checkGroupAccessForList(
	ctx context.Context,
	collegeID int64,
	actorID int64,
	actorRole string,
	groupID int64,
) error {
	if collegeID <= 0 || actorID <= 0 || groupID <= 0 {
		return ErrInvalidInput
	}

	if actorRole == "college_admin" {
		const query = `SELECT EXISTS (
			SELECT 1 FROM groups
			WHERE id = $1 AND college_id = $2 AND is_active = TRUE
		)`
		var exists bool
		if err := s.db.QueryRow(ctx, query, groupID, collegeID).Scan(&exists); err != nil {
			return fmt.Errorf("check access period list group: %w", err)
		}
		if !exists {
			return ErrGroupNotFound
		}
		return nil
	}

	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM groups g
			JOIN group_memberships gm
			  ON gm.group_id = g.id
			 AND gm.college_id = g.college_id
			 AND gm.user_id = $3
			WHERE g.id = $1
			  AND g.college_id = $2
			  AND g.is_active = TRUE
			  AND gm.membership_role = $4
		)
	`

	var member bool
	if err := s.db.QueryRow(
		ctx, query, groupID, collegeID, actorID, actorRole,
	).Scan(&member); err != nil {
		return fmt.Errorf("check access period list membership: %w", err)
	}
	if !member {
		return ErrForbidden
	}

	return nil
}

func (s *Service) IsOpen(
	ctx context.Context,
	collegeID int64,
	subgroupID int64,
	at time.Time,
) (bool, error) {
	if collegeID <= 0 || subgroupID <= 0 || at.IsZero() {
		return false, ErrInvalidInput
	}

	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM subgroup_access_periods
			WHERE college_id = $1
			  AND subgroup_id = $2
			  AND starts_at <= $3
			  AND ends_at > $3
		)
	`

	var open bool
	if err := s.db.QueryRow(
		ctx, query, collegeID, subgroupID, at,
	).Scan(&open); err != nil {
		return false, fmt.Errorf("check subgroup access period: %w", err)
	}

	return open, nil
}

func isExclusionViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23P01"
}
