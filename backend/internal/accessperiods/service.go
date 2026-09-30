package accessperiods

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrInvalidInput         = errors.New("invalid access period input")
	ErrForbidden            = errors.New("forbidden")
	ErrSubgroupNotFound     = errors.New("subgroup not found")
	ErrAccessPeriodOverlap  = errors.New("access period overlaps an existing period")
	ErrAccessPeriodNotFound = errors.New("access period not found")
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

type AccessPeriod struct {
	ID         int64
	SubgroupID int64
	GroupID    int64
	CollegeID  int64
	StartsAt   string
	EndsAt     string
	CreatedBy  int64
	CreatedAt  string
}

type CreateInput struct {
	StartsAt string
	EndsAt   string
}

// checkAuthorization verifies that the actor is authorized to
// manage access periods for the requested subgroup.
//
// College admins can manage access periods throughout their college.
//
// Faculty members must have an active assignment to the requested
// subgroup.
//
// Students and all other roles are forbidden.
func (s *Service) checkAuthorization(
	ctx context.Context,
	collegeID int64,
	actorID int64,
	actorRole string,
	groupID int64,
	subgroupID int64,
) error {
	if collegeID <= 0 ||
		actorID <= 0 ||
		groupID <= 0 ||
		subgroupID <= 0 {
		return ErrInvalidInput
	}

	if actorRole != "college_admin" && actorRole != "faculty" {
		return ErrForbidden
	}

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
		return fmt.Errorf(
			"check access period subgroup: %w",
			err,
		)
	}

	if !subgroupExists {
		return ErrSubgroupNotFound
	}

	// College admins can manage access periods throughout
	// their own college.
	if actorRole == "college_admin" {
		return nil
	}

	const assignmentQuery = `
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

	err = s.db.QueryRow(
		ctx,
		assignmentQuery,
		collegeID,
		subgroupID,
		actorID,
	).Scan(&assigned)

	if err != nil {
		return fmt.Errorf(
			"check access period faculty assignment: %w",
			err,
		)
	}

	if !assigned {
		return ErrForbidden
	}

	return nil
}

// Create creates a new access period for an authorized actor.
//
// Access periods are append-only. Reopening a subgroup creates
// a new access period instead of modifying an existing period.
//
// PostgreSQL prevents overlapping periods for the same subgroup.
func (s *Service) Create(
	ctx context.Context,
	collegeID int64,
	actorID int64,
	actorRole string,
	groupID int64,
	subgroupID int64,
	input CreateInput,
) (AccessPeriod, error) {
	if collegeID <= 0 ||
		actorID <= 0 ||
		groupID <= 0 ||
		subgroupID <= 0 {
		return AccessPeriod{}, ErrInvalidInput
	}

	if input.StartsAt == "" ||
		input.EndsAt == "" ||
		input.StartsAt >= input.EndsAt {
		return AccessPeriod{}, ErrInvalidInput
	}

	if err := s.checkAuthorization(
		ctx,
		collegeID,
		actorID,
		actorRole,
		groupID,
		subgroupID,
	); err != nil {
		return AccessPeriod{}, err
	}

	const insertQuery = `
		INSERT INTO subgroup_access_periods (
			subgroup_id,
			college_id,
			starts_at,
			ends_at,
			created_by
		)
		VALUES (
			$1,
			$2,
			$3::timestamptz,
			$4::timestamptz,
			$5
		)
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
		insertQuery,
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
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) &&
			pgErr.Code == "23P01" &&
			pgErr.ConstraintName == "subgroup_access_periods_no_overlap" {
			return AccessPeriod{}, ErrAccessPeriodOverlap
		}

		return AccessPeriod{}, fmt.Errorf(
			"create access period: insert: %w",
			err,
		)
	}

	// group_id is not stored in subgroup_access_periods.
	// It was already validated by checkAuthorization and is
	// therefore added to the response object here.
	period.GroupID = groupID

	return period, nil
}

// List returns all access periods for an authorized actor.
//
// Results are ordered chronologically by start time.
func (s *Service) List(
	ctx context.Context,
	collegeID int64,
	actorID int64,
	actorRole string,
	groupID int64,
	subgroupID int64,
) ([]AccessPeriod, error) {
	if collegeID <= 0 ||
		actorID <= 0 ||
		groupID <= 0 ||
		subgroupID <= 0 {
		return nil, ErrInvalidInput
	}

	if err := s.checkAuthorization(
		ctx,
		collegeID,
		actorID,
		actorRole,
		groupID,
		subgroupID,
	); err != nil {
		return nil, err
	}

	const listQuery = `
		SELECT
			id,
			subgroup_id,
			college_id,
			starts_at::text,
			ends_at::text,
			created_by,
			created_at::text
		FROM subgroup_access_periods
		WHERE subgroup_id = $1
		  AND college_id = $2
		ORDER BY starts_at ASC, id ASC
	`

	rows, err := s.db.Query(
		ctx,
		listQuery,
		subgroupID,
		collegeID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"list access periods: query: %w",
			err,
		)
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
			return nil, fmt.Errorf(
				"list access periods: scan: %w",
				err,
			)
		}

		// group_id is not stored in subgroup_access_periods.
		// It comes from the validated request context.
		period.GroupID = groupID

		result = append(result, period)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"list access periods: rows: %w",
			err,
		)
	}

	return result, nil
}

// Current returns the access period containing the supplied instant.
//
// Access periods use half-open intervals:
//
// [starts_at, ends_at)
//
// Therefore:
//
// starts_at <= time < ends_at
func (s *Service) Current(
	ctx context.Context,
	collegeID int64,
	groupID int64,
	subgroupID int64,
	at string,
) (AccessPeriod, error) {
	if collegeID <= 0 ||
		groupID <= 0 ||
		subgroupID <= 0 ||
		at == "" {
		return AccessPeriod{}, ErrInvalidInput
	}

	const query = `
		SELECT
			ap.id,
			ap.subgroup_id,
			ap.college_id,
			ap.starts_at::text,
			ap.ends_at::text,
			ap.created_by,
			ap.created_at::text
		FROM subgroup_access_periods ap
		JOIN subgroups s
		  ON s.id = ap.subgroup_id
		 AND s.college_id = ap.college_id
		WHERE ap.subgroup_id = $1
		  AND ap.college_id = $2
		  AND s.group_id = $3
		  AND s.is_active = TRUE
		  AND ap.starts_at <= $4::timestamptz
		  AND ap.ends_at > $4::timestamptz
		ORDER BY ap.starts_at DESC, ap.id DESC
		LIMIT 1
	`

	var period AccessPeriod

	err := s.db.QueryRow(
		ctx,
		query,
		subgroupID,
		collegeID,
		groupID,
		at,
	).Scan(
		&period.ID,
		&period.SubgroupID,
		&period.CollegeID,
		&period.StartsAt,
		&period.EndsAt,
		&period.CreatedBy,
		&period.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return AccessPeriod{}, ErrAccessPeriodNotFound
	}

	if err != nil {
		return AccessPeriod{}, fmt.Errorf(
			"get current access period: %w",
			err,
		)
	}

	// group_id is not stored in subgroup_access_periods.
	// It is already validated by the query above.
	period.GroupID = groupID

	return period, nil
}

// AuthorizeStudentAccess verifies that a student is allowed to access
// a subgroup at the specified time.
//
// A student must:
//   - belong to the requested college
//   - be a student member of the requested group
//   - access an active subgroup belonging to that group
//   - have a currently active access period
//
// Access periods use half-open intervals:
// [starts_at, ends_at)
func (s *Service) AuthorizeStudentAccess(
	ctx context.Context,
	collegeID int64,
	studentID int64,
	groupID int64,
	subgroupID int64,
	at string,
) error {
	if collegeID <= 0 ||
		studentID <= 0 ||
		groupID <= 0 ||
		subgroupID <= 0 ||
		at == "" {
		return ErrInvalidInput
	}

	// Verify that the student is a member of the requested group.
	const membershipQuery = `
		SELECT EXISTS (
			SELECT 1
			FROM group_memberships
			WHERE college_id = $1
			  AND group_id = $2
			  AND user_id = $3
			  AND membership_role = 'student'
		)
	`

	var isMember bool

	err := s.db.QueryRow(
		ctx,
		membershipQuery,
		collegeID,
		groupID,
		studentID,
	).Scan(&isMember)

	if err != nil {
		return fmt.Errorf(
			"authorize student subgroup access: check membership: %w",
			err,
		)
	}

	if !isMember {
		return ErrForbidden
	}

	// Verify that the subgroup belongs to the requested group and college
	// and has not been archived.
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

	err = s.db.QueryRow(
		ctx,
		subgroupQuery,
		subgroupID,
		groupID,
		collegeID,
	).Scan(&subgroupExists)

	if err != nil {
		return fmt.Errorf(
			"authorize student subgroup access: check subgroup: %w",
			err,
		)
	}

	if !subgroupExists {
		return ErrForbidden
	}

	// Verify that the requested time falls inside an access period.
	//
	// The interval is half-open:
	//
	//     starts_at <= at < ends_at
	//
	// Therefore:
	// - exactly at starts_at -> allowed
	// - exactly at ends_at   -> denied
	const accessPeriodQuery = `
		SELECT EXISTS (
			SELECT 1
			FROM subgroup_access_periods
			WHERE subgroup_id = $1
			  AND college_id = $2
			  AND starts_at <= $3::timestamptz
			  AND ends_at > $3::timestamptz
		)
	`

	var hasAccessPeriod bool

	err = s.db.QueryRow(
		ctx,
		accessPeriodQuery,
		subgroupID,
		collegeID,
		at,
	).Scan(&hasAccessPeriod)

	if err != nil {
		return fmt.Errorf(
			"authorize student subgroup access: check access period: %w",
			err,
		)
	}

	if !hasAccessPeriod {
		return ErrForbidden
	}

	return nil
}
