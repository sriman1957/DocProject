package accessperiods

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type mockDB struct {
	queryRowFn func(ctx context.Context, sql string, args ...any) pgx.Row
	execFn     func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (m *mockDB) QueryRow(
	ctx context.Context,
	sql string,
	args ...any,
) pgx.Row {
	return m.queryRowFn(ctx, sql, args...)
}

func (m *mockDB) Query(
	ctx context.Context,
	sql string,
	args ...any,
) (pgx.Rows, error) {
	if m.execFn == nil {
		return nil, errors.New("unexpected Query call")
	}

	return m.execFn(ctx, sql, args...)
}

type mockRow struct {
	values []any
	err    error
}

func (r mockRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}

	if len(dest) != len(r.values) {
		return errors.New("destination count mismatch")
	}

	for i := range dest {
		switch d := dest[i].(type) {
		case *bool:
			v, ok := r.values[i].(bool)
			if !ok {
				return errors.New("expected bool value")
			}

			*d = v

		case *int64:
			v, ok := r.values[i].(int64)
			if !ok {
				return errors.New("expected int64 value")
			}

			*d = v

		case *string:
			switch v := r.values[i].(type) {
			case string:
				*d = v

			case time.Time:
				*d = v.Format(time.RFC3339Nano)

			default:
				return errors.New("expected string value")
			}

		case *time.Time:
			switch v := r.values[i].(type) {
			case time.Time:
				*d = v

			case string:
				parsed, err := time.Parse(
					time.RFC3339Nano,
					v,
				)
				if err != nil {
					return err
				}

				*d = parsed

			default:
				return errors.New("expected time.Time value")
			}

		default:
			return errors.New("unsupported scan destination")
		}
	}

	return nil
}

func TestCreateAccessPeriodAuthorization_AdminAllowed(t *testing.T) {
	t.Parallel()

	startsAt := "2026-09-29T10:00:00Z"
	endsAt := "2026-09-29T12:00:00Z"
	createdAt := "2026-09-29T09:00:00Z"

	var queryCount int

	db := &mockDB{
		queryRowFn: func(
			ctx context.Context,
			sql string,
			args ...any,
		) pgx.Row {
			queryCount++

			switch queryCount {
			case 1:
				// Active subgroup authorization check.
				return mockRow{
					values: []any{true},
				}

			case 2:
				// INSERT ... RETURNING.
				//
				// group_id is NOT stored in
				// subgroup_access_periods.
				//
				// Returned columns:
				// id
				// subgroup_id
				// college_id
				// starts_at
				// ends_at
				// created_by
				// created_at
				return mockRow{
					values: []any{
						int64(1),
						int64(200),
						int64(7),
						startsAt,
						endsAt,
						int64(42),
						createdAt,
					},
				}

			default:
				return mockRow{
					err: errors.New(
						"unexpected QueryRow call",
					),
				}
			}
		},
	}

	service := NewService(db)

	result, err := service.Create(
		context.Background(),
		7,
		42,
		"college_admin",
		100,
		200,
		CreateInput{
			StartsAt: startsAt,
			EndsAt:   endsAt,
		},
	)

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if result.ID != 1 {
		t.Errorf(
			"expected ID 1, got %d",
			result.ID,
		)
	}

	if result.CollegeID != 7 {
		t.Errorf(
			"expected college ID 7, got %d",
			result.CollegeID,
		)
	}

	if result.GroupID != 100 {
		t.Errorf(
			"expected group ID 100, got %d",
			result.GroupID,
		)
	}

	if result.SubgroupID != 200 {
		t.Errorf(
			"expected subgroup ID 200, got %d",
			result.SubgroupID,
		)
	}

	if result.StartsAt != startsAt {
		t.Errorf(
			"expected starts_at %q, got %q",
			startsAt,
			result.StartsAt,
		)
	}

	if result.EndsAt != endsAt {
		t.Errorf(
			"expected ends_at %q, got %q",
			endsAt,
			result.EndsAt,
		)
	}

	if result.CreatedBy != 42 {
		t.Errorf(
			"expected created_by 42, got %d",
			result.CreatedBy,
		)
	}
}

func TestCreateAccessPeriodAuthorization_FacultyAllowedWhenAssigned(
	t *testing.T,
) {
	t.Parallel()

	startsAt := "2026-09-29T10:00:00Z"
	endsAt := "2026-09-29T12:00:00Z"
	createdAt := "2026-09-29T09:00:00Z"

	var queryCount int

	db := &mockDB{
		queryRowFn: func(
			ctx context.Context,
			sql string,
			args ...any,
		) pgx.Row {
			queryCount++

			switch queryCount {
			case 1:
				// Active subgroup check.
				return mockRow{
					values: []any{true},
				}

			case 2:
				// Active faculty assignment check.
				return mockRow{
					values: []any{true},
				}

			case 3:
				// INSERT ... RETURNING.
				return mockRow{
					values: []any{
						int64(1),
						int64(200),
						int64(7),
						startsAt,
						endsAt,
						int64(42),
						createdAt,
					},
				}

			default:
				return mockRow{
					err: errors.New(
						"unexpected QueryRow call",
					),
				}
			}
		},
	}

	service := NewService(db)

	result, err := service.Create(
		context.Background(),
		7,
		42,
		"faculty",
		100,
		200,
		CreateInput{
			StartsAt: startsAt,
			EndsAt:   endsAt,
		},
	)

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if result.ID != 1 {
		t.Errorf(
			"expected ID 1, got %d",
			result.ID,
		)
	}

	if result.CollegeID != 7 {
		t.Errorf(
			"expected college ID 7, got %d",
			result.CollegeID,
		)
	}

	if result.GroupID != 100 {
		t.Errorf(
			"expected group ID 100, got %d",
			result.GroupID,
		)
	}

	if result.SubgroupID != 200 {
		t.Errorf(
			"expected subgroup ID 200, got %d",
			result.SubgroupID,
		)
	}

	if result.StartsAt != startsAt {
		t.Errorf(
			"expected starts_at %q, got %q",
			startsAt,
			result.StartsAt,
		)
	}

	if result.EndsAt != endsAt {
		t.Errorf(
			"expected ends_at %q, got %q",
			endsAt,
			result.EndsAt,
		)
	}

	if result.CreatedBy != 42 {
		t.Errorf(
			"expected created_by 42, got %d",
			result.CreatedBy,
		)
	}
}

func TestCreateAccessPeriodAuthorization_FacultyNotAssigned(
	t *testing.T,
) {
	t.Parallel()

	var queryCount int

	db := &mockDB{
		queryRowFn: func(
			ctx context.Context,
			sql string,
			args ...any,
		) pgx.Row {
			queryCount++

			switch queryCount {
			case 1:
				// Active subgroup.
				return mockRow{
					values: []any{true},
				}

			case 2:
				// Faculty assignment does not exist.
				return mockRow{
					values: []any{false},
				}

			default:
				return mockRow{
					err: errors.New(
						"unexpected QueryRow call",
					),
				}
			}
		},
	}

	service := NewService(db)

	_, err := service.Create(
		context.Background(),
		7,
		42,
		"faculty",
		100,
		200,
		CreateInput{
			StartsAt: "2026-09-29T10:00:00Z",
			EndsAt:   "2026-09-29T12:00:00Z",
		},
	)

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf(
			"expected ErrForbidden, got %v",
			err,
		)
	}
}

func TestCreateAccessPeriodAuthorization_StudentForbidden(
	t *testing.T,
) {
	t.Parallel()

	db := &mockDB{
		queryRowFn: func(
			ctx context.Context,
			sql string,
			args ...any,
		) pgx.Row {
			return mockRow{
				values: []any{true},
			}
		},
	}

	service := NewService(db)

	_, err := service.Create(
		context.Background(),
		7,
		42,
		"student",
		100,
		200,
		CreateInput{
			StartsAt: "2026-09-29T10:00:00Z",
			EndsAt:   "2026-09-29T12:00:00Z",
		},
	)

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf(
			"expected ErrForbidden, got %v",
			err,
		)
	}
}

func TestCreateAccessPeriodAuthorization_UnknownRoleForbidden(
	t *testing.T,
) {
	t.Parallel()

	db := &mockDB{
		queryRowFn: func(
			ctx context.Context,
			sql string,
			args ...any,
		) pgx.Row {
			return mockRow{
				values: []any{true},
			}
		},
	}

	service := NewService(db)

	_, err := service.Create(
		context.Background(),
		7,
		42,
		"unknown",
		100,
		200,
		CreateInput{
			StartsAt: "2026-09-29T10:00:00Z",
			EndsAt:   "2026-09-29T12:00:00Z",
		},
	)

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf(
			"expected ErrForbidden, got %v",
			err,
		)
	}
}

func TestCreateAccessPeriodAuthorization_SubgroupNotFound(
	t *testing.T,
) {
	t.Parallel()

	db := &mockDB{
		queryRowFn: func(
			ctx context.Context,
			sql string,
			args ...any,
		) pgx.Row {
			return mockRow{
				values: []any{false},
			}
		},
	}

	service := NewService(db)

	_, err := service.Create(
		context.Background(),
		7,
		42,
		"college_admin",
		100,
		200,
		CreateInput{
			StartsAt: "2026-09-29T10:00:00Z",
			EndsAt:   "2026-09-29T12:00:00Z",
		},
	)

	if !errors.Is(err, ErrSubgroupNotFound) {
		t.Fatalf(
			"expected ErrSubgroupNotFound, got %v",
			err,
		)
	}
}

func TestCreateAccessPeriodAuthorization_QueryError(
	t *testing.T,
) {
	t.Parallel()

	expectedErr := errors.New("database failure")

	db := &mockDB{
		queryRowFn: func(
			ctx context.Context,
			sql string,
			args ...any,
		) pgx.Row {
			return mockRow{
				err: expectedErr,
			}
		},
	}

	service := NewService(db)

	_, err := service.Create(
		context.Background(),
		7,
		42,
		"college_admin",
		100,
		200,
		CreateInput{
			StartsAt: "2026-09-29T10:00:00Z",
			EndsAt:   "2026-09-29T12:00:00Z",
		},
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected database error, got %v",
			err,
		)
	}
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	service := NewService(nil)

	tests := []CreateInput{
		{},
		{
			StartsAt: "2026-10-01T10:00:00Z",
		},
		{
			EndsAt: "2026-10-01T10:00:00Z",
		},
		{
			StartsAt: "2026-10-01T11:00:00Z",
			EndsAt:   "2026-10-01T10:00:00Z",
		},
	}

	for _, input := range tests {
		_, err := service.Create(
			context.Background(),
			1,
			1,
			"college_admin",
			1,
			1,
			input,
		)

		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf(
				"input %+v: expected ErrInvalidInput, got %v",
				input,
				err,
			)
		}
	}
}

func TestCreateRejectsInvalidIDs(t *testing.T) {
	t.Parallel()

	service := NewService(nil)

	_, err := service.Create(
		context.Background(),
		0,
		1,
		"college_admin",
		1,
		1,
		CreateInput{
			StartsAt: "2026-10-01T10:00:00Z",
			EndsAt:   "2026-10-01T11:00:00Z",
		},
	)

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf(
			"expected ErrInvalidInput, got %v",
			err,
		)
	}
}

func TestCreateMapsMissingSubgroup(t *testing.T) {
	t.Parallel()

	service := NewService(&mockDB{
		queryRowFn: func(
			ctx context.Context,
			sql string,
			args ...any,
		) pgx.Row {
			return mockRow{
				values: []any{false},
			}
		},
	})

	_, err := service.Create(
		context.Background(),
		1,
		1,
		"college_admin",
		10,
		20,
		CreateInput{
			StartsAt: "2026-10-01T10:00:00Z",
			EndsAt:   "2026-10-01T11:00:00Z",
		},
	)

	if !errors.Is(err, ErrSubgroupNotFound) {
		t.Fatalf(
			"expected ErrSubgroupNotFound, got %v",
			err,
		)
	}
}

func TestListRejectsInvalidIDs(t *testing.T) {
	t.Parallel()

	service := NewService(nil)

	_, err := service.List(
		context.Background(),
		1,
		1,
		"college_admin",
		1,
		0,
	)

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf(
			"expected ErrInvalidInput, got %v",
			err,
		)
	}
}

func TestCurrentRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	service := NewService(nil)

	_, err := service.Current(
		context.Background(),
		1,
		1,
		1,
		"",
	)

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf(
			"expected ErrInvalidInput, got %v",
			err,
		)
	}
}

func TestCurrentMapsNoActivePeriod(t *testing.T) {
	t.Parallel()

	service := NewService(&mockDB{
		queryRowFn: func(
			ctx context.Context,
			sql string,
			args ...any,
		) pgx.Row {
			return mockRow{
				err: pgx.ErrNoRows,
			}
		},
	})

	_, err := service.Current(
		context.Background(),
		1,
		1,
		1,
		"2026-10-01T10:30:00Z",
	)

	if !errors.Is(err, ErrAccessPeriodNotFound) {
		t.Fatalf(
			"expected ErrAccessPeriodNotFound, got %v",
			err,
		)
	}
}

func TestServiceAuthorizeStudentAccess_AllowsStudentInsidePeriod(t *testing.T) {
	t.Parallel()

	var queryCount int

	db := &mockDB{
		queryRowFn: func(
			ctx context.Context,
			sql string,
			args ...any,
		) pgx.Row {
			queryCount++

			switch queryCount {
			case 1:
				return mockRow{values: []any{true}}
			case 2:
				return mockRow{values: []any{true}}
			case 3:
				return mockRow{values: []any{true}}
			default:
				return mockRow{err: errors.New("unexpected QueryRow call")}
			}
		},
	}

	service := NewService(db)

	err := service.AuthorizeStudentAccess(
		context.Background(),
		1,
		20,
		10,
		100,
		"2026-09-29T10:00:00Z",
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if queryCount != 3 {
		t.Fatalf("expected 3 QueryRow calls, got %d", queryCount)
	}
}

func TestServiceAuthorizeStudentAccess_StudentNotMemberForbidden(t *testing.T) {
	t.Parallel()

	var queryCount int

	db := &mockDB{
		queryRowFn: func(
			ctx context.Context,
			sql string,
			args ...any,
		) pgx.Row {
			queryCount++

			if queryCount != 1 {
				return mockRow{err: errors.New("unexpected QueryRow call")}
			}

			return mockRow{values: []any{false}}
		},
	}

	service := NewService(db)

	err := service.AuthorizeStudentAccess(
		context.Background(),
		1,
		20,
		10,
		100,
		"2026-09-29T10:00:00Z",
	)

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}

	if queryCount != 1 {
		t.Fatalf("expected 1 QueryRow call, got %d", queryCount)
	}
}

func TestServiceAuthorizeStudentAccess_SubgroupForbidden(t *testing.T) {
	t.Parallel()

	var queryCount int

	db := &mockDB{
		queryRowFn: func(
			ctx context.Context,
			sql string,
			args ...any,
		) pgx.Row {
			queryCount++

			switch queryCount {
			case 1:
				return mockRow{values: []any{true}}
			case 2:
				return mockRow{values: []any{false}}
			default:
				return mockRow{err: errors.New("unexpected QueryRow call")}
			}
		},
	}

	service := NewService(db)

	err := service.AuthorizeStudentAccess(
		context.Background(),
		1,
		20,
		10,
		100,
		"2026-09-29T10:00:00Z",
	)

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}

	if queryCount != 2 {
		t.Fatalf("expected 2 QueryRow calls, got %d", queryCount)
	}
}

func TestServiceAuthorizeStudentAccess_NoCurrentPeriodForbidden(t *testing.T) {
	t.Parallel()

	var queryCount int

	db := &mockDB{
		queryRowFn: func(
			ctx context.Context,
			sql string,
			args ...any,
		) pgx.Row {
			queryCount++

			switch queryCount {
			case 1, 2:
				return mockRow{values: []any{true}}
			case 3:
				return mockRow{values: []any{false}}
			default:
				return mockRow{err: errors.New("unexpected QueryRow call")}
			}
		},
	}

	service := NewService(db)

	err := service.AuthorizeStudentAccess(
		context.Background(),
		1,
		20,
		10,
		100,
		"2026-09-29T13:00:00Z",
	)

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}

	if queryCount != 3 {
		t.Fatalf("expected 3 QueryRow calls, got %d", queryCount)
	}
}

func TestServiceAuthorizeStudentAccess_InvalidInput(t *testing.T) {
	t.Parallel()

	service := NewService(nil)

	tests := []struct {
		name       string
		collegeID  int64
		studentID  int64
		groupID    int64
		subgroupID int64
		at         string
	}{
		{
			name:       "invalid college",
			collegeID:  0,
			studentID:  20,
			groupID:    10,
			subgroupID: 100,
			at:         "2026-09-29T10:00:00Z",
		},
		{
			name:       "invalid student",
			collegeID:  1,
			studentID:  0,
			groupID:    10,
			subgroupID: 100,
			at:         "2026-09-29T10:00:00Z",
		},
		{
			name:       "invalid group",
			collegeID:  1,
			studentID:  20,
			groupID:    0,
			subgroupID: 100,
			at:         "2026-09-29T10:00:00Z",
		},
		{
			name:       "invalid subgroup",
			collegeID:  1,
			studentID:  20,
			groupID:    10,
			subgroupID: 0,
			at:         "2026-09-29T10:00:00Z",
		},
		{
			name:       "missing time",
			collegeID:  1,
			studentID:  20,
			groupID:    10,
			subgroupID: 100,
			at:         "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.AuthorizeStudentAccess(
				context.Background(),
				tt.collegeID,
				tt.studentID,
				tt.groupID,
				tt.subgroupID,
				tt.at,
			)

			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestServiceAuthorizeStudentAccess_MembershipQueryError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("database failure")

	db := &mockDB{
		queryRowFn: func(
			ctx context.Context,
			sql string,
			args ...any,
		) pgx.Row {
			return mockRow{err: expectedErr}
		},
	}

	service := NewService(db)

	err := service.AuthorizeStudentAccess(
		context.Background(),
		1,
		20,
		10,
		100,
		"2026-09-29T10:00:00Z",
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected database error, got %v", err)
	}
}

func TestServiceAuthorizeStudentAccess_SubgroupQueryError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("database failure")
	var queryCount int

	db := &mockDB{
		queryRowFn: func(
			ctx context.Context,
			sql string,
			args ...any,
		) pgx.Row {
			queryCount++

			switch queryCount {
			case 1:
				return mockRow{values: []any{true}}
			case 2:
				return mockRow{err: expectedErr}
			default:
				return mockRow{err: errors.New("unexpected QueryRow call")}
			}
		},
	}

	service := NewService(db)

	err := service.AuthorizeStudentAccess(
		context.Background(),
		1,
		20,
		10,
		100,
		"2026-09-29T10:00:00Z",
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected database error, got %v", err)
	}
}

func TestServiceAuthorizeStudentAccess_AccessPeriodQueryError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("database failure")
	var queryCount int

	db := &mockDB{
		queryRowFn: func(
			ctx context.Context,
			sql string,
			args ...any,
		) pgx.Row {
			queryCount++

			switch queryCount {
			case 1, 2:
				return mockRow{values: []any{true}}
			case 3:
				return mockRow{err: expectedErr}
			default:
				return mockRow{err: errors.New("unexpected QueryRow call")}
			}
		},
	}

	service := NewService(db)

	err := service.AuthorizeStudentAccess(
		context.Background(),
		1,
		20,
		10,
		100,
		"2026-09-29T10:00:00Z",
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected database error, got %v", err)
	}
}
