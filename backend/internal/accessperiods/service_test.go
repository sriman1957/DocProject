package accessperiods

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type fakeRow struct {
	scan func(dest ...any) error
}

func (r fakeRow) Scan(dest ...any) error {
	return r.scan(dest...)
}

type fakeQuerier struct {
	rows int
}

func (f *fakeQuerier) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	f.rows++
	return fakeRow{scan: func(dest ...any) error {
		if len(dest) == 1 {
			if v, ok := dest[0].(*bool); ok {
				*v = true
				return nil
			}
		}
		return nil
	}}
}

func (f *fakeQuerier) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}

func TestCreateRejectsInvalidTimeRange(t *testing.T) {
	service := NewService(&fakeQuerier{})
	now := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)

	_, err := service.Create(
		context.Background(),
		1, 2, "college_admin", 3, 4,
		CreateInput{StartsAt: now, EndsAt: now},
	)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateRejectsUnsupportedRole(t *testing.T) {
	service := NewService(&fakeQuerier{})
	now := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)

	_, err := service.Create(
		context.Background(),
		1, 2, "student", 3, 4,
		CreateInput{StartsAt: now, EndsAt: now.Add(time.Hour)},
	)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestListRejectsInvalidIDs(t *testing.T) {
	service := NewService(&fakeQuerier{})

	_, err := service.List(
		context.Background(),
		1, 2, "student", 3, 0,
	)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestIsOpenRejectsInvalidInput(t *testing.T) {
	service := NewService(&fakeQuerier{})

	_, err := service.IsOpen(
		context.Background(),
		0, 4, time.Now(),
	)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateChecksGroupAndSubgroupAndAssignment(t *testing.T) {
	service := NewService(&fakeQuerier{})
	now := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)

	_, err := service.Create(
		context.Background(),
		1, 2, "faculty", 3, 4,
		CreateInput{StartsAt: now, EndsAt: now.Add(time.Hour)},
	)
	if err != nil {
		t.Fatalf("expected authorization checks to pass with positive fake rows, got %v", err)
	}
}
