package accessperiods

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

type mockRow struct {
	err error
}

func (r mockRow) Scan(...any) error {
	return r.err
}

type mockDB struct {
	rowErrors []error
	calls     int
}

func (m *mockDB) QueryRow(
	context.Context,
	string,
	...any,
) pgx.Row {
	err := error(nil)

	if m.calls < len(m.rowErrors) {
		err = m.rowErrors[m.calls]
	}

	m.calls++

	return mockRow{err: err}
}

func (m *mockDB) Query(
	context.Context,
	string,
	...any,
) (pgx.Rows, error) {
	return nil, errors.New("unexpected query")
}

func TestCreateRejectsInvalidInput(t *testing.T) {
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
			1,
			1,
			input,
		)

		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("input %+v: expected ErrInvalidInput, got %v", input, err)
		}
	}
}

func TestCreateRejectsInvalidIDs(t *testing.T) {
	service := NewService(nil)

	_, err := service.Create(
		context.Background(),
		0,
		1,
		1,
		1,
		CreateInput{
			StartsAt: "2026-10-01T10:00:00Z",
			EndsAt:   "2026-10-01T11:00:00Z",
		},
	)

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateMapsMissingSubgroup(t *testing.T) {
	service := NewService(&mockDB{
		rowErrors: []error{pgx.ErrNoRows},
	})

	_, err := service.Create(
		context.Background(),
		1,
		1,
		1,
		10,
		CreateInput{
			StartsAt: "2026-10-01T10:00:00Z",
			EndsAt:   "2026-10-01T11:00:00Z",
		},
	)

	if !errors.Is(err, ErrSubgroupNotFound) {
		t.Fatalf("expected ErrSubgroupNotFound, got %v", err)
	}
}

func TestListRejectsInvalidIDs(t *testing.T) {
	service := NewService(nil)

	_, err := service.List(context.Background(), 1, 1, 0)

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCurrentRejectsInvalidInput(t *testing.T) {
	service := NewService(nil)

	_, err := service.Current(
		context.Background(),
		1,
		1,
		1,
		"",
	)

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCurrentMapsNoActivePeriod(t *testing.T) {
	service := NewService(&mockDB{
		rowErrors: []error{pgx.ErrNoRows},
	})

	_, err := service.Current(
		context.Background(),
		1,
		1,
		1,
		"2026-10-01T10:30:00Z",
	)

	if !errors.Is(err, ErrAccessPeriodNotFound) {
		t.Fatalf("expected ErrAccessPeriodNotFound, got %v", err)
	}
}
