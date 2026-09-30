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
	ErrGroupNotFound        = errors.New("group not found")
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
	return &Service{db: database}
}

type AccessPeriod struct {
	ID         int64  `json:"id"`
	SubgroupID int64  `json:"subgroup_id"`
	CollegeID  int64  `json:"college_id"`
	StartsAt   string `json:"starts_at"`
	EndsAt     string `json:"ends_at"`
	CreatedBy  int64  `json:"created_by"`
	CreatedAt  string `json:"created_at"`
}

type CreateInput struct {
	StartsAt string
	EndsAt   string
}

// Create creates a new access period for an active subgroup.
//
// Access periods are append-only. Reopening a subgroup creates a new
// period rather than modifying an existing one. PostgreSQL enforces
// that periods for the same subgroup cannot overlap.
func (s *Service) Create(
	ctx context.Context,
	collegeID int64,
	createdBy int64,
	groupID int64,
	subgroupID int64,
	input CreateInput,
) (AccessPeriod, error) {
	if collegeID <= 0 || createdBy <= 0 || groupID <= 0 || subgroupID <= 0 {
		return AccessPeriod{}, ErrInvalidInput
	}

	if input.StartsAt == "" || input.EndsAt == "" ||
		input.StartsAt >= input.EndsAt {
		return AccessPeriod{}, ErrInvalidInput
	}

	const subgroupQuery = `
		SELECT id
		FROM subgroups
		WHERE id = $1
		  AND group_id = $2
		  AND college_id = $3
		  AND is_active = TRUE
	`

	var foundSubgroupID int64

	err := s.db.QueryRow(
		ctx,
		subgroupQuery,
		subgroupID,
		groupID,
		collegeID,
	).Scan(&foundSubgroupID)

	if errors.Is(err, pgx.ErrNoRows) {
		return AccessPeriod{}, ErrSubgroupNotFound
	}

	if err != nil {
		return AccessPeriod{}, fmt.Errorf(
			"create access period: check subgroup: %w",
			err,
		)
	}

	const insertQuery = `
		INSERT INTO subgroup_access_periods (
			subgroup_id,
			college_id,
			starts_at,
			ends_at,
			created_by
		)
		VALUES ($1, $2, $3::timestamptz, $4::timestamptz, $5)
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

	err = s.db.QueryRow(
		ctx,
		insertQuery,
		subgroupID,
		collegeID,
		input.StartsAt,
		input.EndsAt,
		createdBy,
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

	return period, nil
}

// List returns all access periods for an active subgroup in chronological order.
func (s *Service) List(
	ctx context.Context,
	collegeID int64,
	groupID int64,
	subgroupID int64,
) ([]AccessPeriod, error) {
	if collegeID <= 0 || groupID <= 0 || subgroupID <= 0 {
		return nil, ErrInvalidInput
	}

	const subgroupQuery = `
		SELECT id
		FROM subgroups
		WHERE id = $1
		  AND group_id = $2
		  AND college_id = $3
		  AND is_active = TRUE
	`

	var foundSubgroupID int64

	err := s.db.QueryRow(
		ctx,
		subgroupQuery,
		subgroupID,
		groupID,
		collegeID,
	).Scan(&foundSubgroupID)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSubgroupNotFound
	}

	if err != nil {
		return nil, fmt.Errorf(
			"list access periods: check subgroup: %w",
			err,
		)
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

	rows, err := s.db.Query(ctx, listQuery, subgroupID, collegeID)
	if err != nil {
		return nil, fmt.Errorf("list access periods: query: %w", err)
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
			return nil, fmt.Errorf("list access periods: scan: %w", err)
		}

		result = append(result, period)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list access periods: rows: %w", err)
	}

	return result, nil
}

// Current returns the access period containing the supplied instant.
// Periods use half-open intervals: [starts_at, ends_at).
func (s *Service) Current(
	ctx context.Context,
	collegeID int64,
	groupID int64,
	subgroupID int64,
	at string,
) (AccessPeriod, error) {
	if collegeID <= 0 || groupID <= 0 || subgroupID <= 0 || at == "" {
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

	return period, nil
}
