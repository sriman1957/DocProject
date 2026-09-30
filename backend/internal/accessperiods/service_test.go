package accessperiods

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeRow struct {
	scan func(dest ...any) error
}

func (r fakeRow) Scan(dest ...any) error {
	return r.scan(dest...)
}

type fakeQuerier struct {
	bools       []bool
	boolIndex   int
	insertError error
	rows        int
}

func (f *fakeQuerier) QueryRow(_ context.Context, query string, _ ...any) pgx.Row {
	f.rows++

	return fakeRow{scan: func(dest ...any) error {
		if f.insertError != nil && len(dest) == 7 {
			return f.insertError
		}

		if len(dest) == 1 {
			v, ok := dest[0].(*bool)
			if !ok {
				return errors.New("expected bool destination")
			}
			if f.boolIndex >= len(f.bools) {
				return errors.New("unexpected boolean query")
			}
			*v = f.bools[f.boolIndex]
			f.boolIndex++
			return nil
		}

		if len(dest) == 7 {
			values := []any{
				int64(10),
				int64(20),
				int64(1),
				"2026-09-30 10:00:00+00",
				"2026-09-30 12:00:00+00",
				int64(2),
				"2026-09-30 09:00:00+00",
			}
			for i := range dest {
				switch d := dest[i].(type) {
				case *int64:
					*d = values[i].(int64)
				case *string:
					*d = values[i].(string)
				default:
					return errors.New("unexpected destination type")
				}
			}
			return nil
		}

		return errors.New("unexpected scan shape")
	}}
}

func (f *fakeQuerier) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}

func validInput() CreateInput {
	start := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)
	return CreateInput{StartsAt: start, EndsAt: start.Add(2 * time.Hour)}
}

func TestCreateRejectsInvalidTimeRange(t *testing.T) {
	service := NewService(&fakeQuerier{})
	input := validInput()
	input.EndsAt = input.StartsAt

	_, err := service.Create(context.Background(), 1, 2, "college_admin", 3, 4, input)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateRejectsUnsupportedRole(t *testing.T) {
	service := NewService(&fakeQuerier{})

	_, err := service.Create(context.Background(), 1, 2, "student", 3, 4, validInput())
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCreateRejectsFacultyWithoutGroupMembership(t *testing.T) {
	service := NewService(&fakeQuerier{bools: []bool{true, false}})

	_, err := service.Create(context.Background(), 1, 2, "faculty", 3, 4, validInput())
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCreateRejectsFacultyWithoutSubgroupAssignment(t *testing.T) {
	service := NewService(&fakeQuerier{bools: []bool{true, true, true, false}})

	_, err := service.Create(context.Background(), 1, 2, "faculty", 3, 4, validInput())
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCreateRejectsMissingGroup(t *testing.T) {
	service := NewService(&fakeQuerier{bools: []bool{false}})

	_, err := service.Create(context.Background(), 1, 2, "college_admin", 3, 4, validInput())
	if !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("expected ErrGroupNotFound, got %v", err)
	}
}

func TestCreateRejectsMissingSubgroup(t *testing.T) {
	service := NewService(&fakeQuerier{bools: []bool{true, false}})

	_, err := service.Create(context.Background(), 1, 2, "college_admin", 3, 4, validInput())
	if !errors.Is(err, ErrSubgroupNotFound) {
		t.Fatalf("expected ErrSubgroupNotFound, got %v", err)
	}
}

func TestCreateReturnsCreatedPeriod(t *testing.T) {
	service := NewService(&fakeQuerier{bools: []bool{true, true}})

	period, err := service.Create(context.Background(), 1, 2, "college_admin", 3, 4, validInput())
	if err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}

	if period.ID != 10 || period.SubgroupID != 20 || period.CollegeID != 1 {
		t.Fatalf("unexpected created period: %+v", period)
	}
	if period.CreatedBy != 2 {
		t.Fatalf("expected created_by 2, got %d", period.CreatedBy)
	}
}

func TestCreateMapsOverlapConstraint(t *testing.T) {
	service := NewService(&fakeQuerier{
		bools: []bool{true, true},
		insertError: &pgconn.PgError{Code: "23P01"},
	})

	_, err := service.Create(context.Background(), 1, 2, "college_admin", 3, 4, validInput())
	if !errors.Is(err, ErrPeriodOverlap) {
		t.Fatalf("expected ErrPeriodOverlap, got %v", err)
	}
}

func TestListRejectsInvalidIDs(t *testing.T) {
	service := NewService(&fakeQuerier{})

	_, err := service.List(context.Background(), 1, 2, "student", 3, 0)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCurrentRejectsInvalidInput(t *testing.T) {
	service := NewService(&fakeQuerier{})

	_, err := service.Current(context.Background(), 1, 2, "student", 3, 4, time.Time{})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestIsOpenRejectsInvalidInput(t *testing.T) {
	service := NewService(&fakeQuerier{})

	_, err := service.IsOpen(context.Background(), 0, 4, time.Now())
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestIsOpenReturnsDatabaseResult(t *testing.T) {
	service := NewService(&fakeQuerier{bools: []bool{true}})

	open, err := service.IsOpen(
		context.Background(),
		1,
		4,
		time.Date(2026, 9, 30, 11, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("expected IsOpen to succeed, got %v", err)
	}
	if !open {
		t.Fatal("expected subgroup to be open")
	}
}
