package branchadmins

import (
	"context"
	"errors"
	"testing"

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
	calls int
	errs  []error
}

func (f *fakeQuerier) QueryRow(
	_ context.Context,
	_ string,
	_ ...any,
) pgx.Row {
	callIndex := f.calls
	f.calls++

	var err error
	if callIndex < len(f.errs) {
		err = f.errs[callIndex]
	}

	return fakeRow{
		scan: func(dest ...any) error {
			if err != nil {
				return err
			}

			switch callIndex {
			case 0:
				*dest[0].(*int64) = 20

			case 1:
				*dest[0].(*int64) = 30

			case 2:
				*dest[0].(*int64) = 40

			case 3:
				*dest[0].(*int64) = 100
				*dest[1].(*int64) = 7
				*dest[2].(*int64) = 20
				*dest[3].(*int64) = 30
				*dest[4].(*int64) = 40
				*dest[5].(*bool) = true
				*dest[6].(*string) = "2026-10-08 09:00:00+00"
				*dest[7].(*string) = "2026-10-08 09:00:00+00"
			}

			return nil
		},
	}
}

func TestAssignSuccess(t *testing.T) {
	database := &fakeQuerier{}
	service := NewService(database)

	assignment, err := service.Assign(
		context.Background(),
		7,
		20,
		30,
		40,
	)
	if err != nil {
		t.Fatalf("Assign returned error: %v", err)
	}

	if assignment.ID != 100 {
		t.Errorf("expected assignment ID 100, got %d", assignment.ID)
	}

	if assignment.CollegeID != 7 {
		t.Errorf("expected college ID 7, got %d", assignment.CollegeID)
	}

	if assignment.BranchID != 20 {
		t.Errorf("expected branch ID 20, got %d", assignment.BranchID)
	}

	if assignment.UserID != 30 {
		t.Errorf("expected user ID 30, got %d", assignment.UserID)
	}

	if assignment.AssignedBy != 40 {
		t.Errorf("expected assigned_by 40, got %d", assignment.AssignedBy)
	}

	if !assignment.IsActive {
		t.Fatal("expected assignment to be active")
	}

	if database.calls != 4 {
		t.Fatalf("expected 4 database calls, got %d", database.calls)
	}
}

func TestAssignRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name       string
		collegeID  int64
		branchID   int64
		userID     int64
		assignedBy int64
	}{
		{
			name:       "invalid college",
			collegeID:  0,
			branchID:   20,
			userID:     30,
			assignedBy: 40,
		},
		{
			name:       "invalid branch",
			collegeID:  7,
			branchID:   0,
			userID:     30,
			assignedBy: 40,
		},
		{
			name:       "invalid user",
			collegeID:  7,
			branchID:   20,
			userID:     0,
			assignedBy: 40,
		},
		{
			name:       "invalid assigner",
			collegeID:  7,
			branchID:   20,
			userID:     30,
			assignedBy: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := &fakeQuerier{}
			service := NewService(database)

			_, err := service.Assign(
				context.Background(),
				test.collegeID,
				test.branchID,
				test.userID,
				test.assignedBy,
			)

			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}

			if database.calls != 0 {
				t.Fatal("database should not be called for invalid input")
			}
		})
	}
}

func TestAssignRejectsMissingBranch(t *testing.T) {
	database := &fakeQuerier{
		errs: []error{
			pgx.ErrNoRows,
		},
	}

	service := NewService(database)

	_, err := service.Assign(
		context.Background(),
		7,
		20,
		30,
		40,
	)

	if !errors.Is(err, ErrBranchNotFound) {
		t.Fatalf("expected ErrBranchNotFound, got %v", err)
	}

	if database.calls != 1 {
		t.Fatalf("expected 1 database call, got %d", database.calls)
	}
}

func TestAssignRejectsMissingFaculty(t *testing.T) {
	database := &fakeQuerier{
		errs: []error{
			nil,
			pgx.ErrNoRows,
		},
	}

	service := NewService(database)

	_, err := service.Assign(
		context.Background(),
		7,
		20,
		30,
		40,
	)

	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}

	if database.calls != 2 {
		t.Fatalf("expected 2 database calls, got %d", database.calls)
	}
}

func TestAssignRejectsInvalidAssigner(t *testing.T) {
	database := &fakeQuerier{
		errs: []error{
			nil,
			nil,
			pgx.ErrNoRows,
		},
	}

	service := NewService(database)

	_, err := service.Assign(
		context.Background(),
		7,
		20,
		30,
		40,
	)

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}

	if database.calls != 3 {
		t.Fatalf("expected 3 database calls, got %d", database.calls)
	}
}

func TestAssignMapsDuplicateAssignment(t *testing.T) {
	database := &fakeQuerier{
		errs: []error{
			nil,
			nil,
			nil,
			&pgconn.PgError{
				Code:           "23505",
				ConstraintName: "uq_branch_admin_assignments_branch_user",
			},
		},
	}

	service := NewService(database)

	_, err := service.Assign(
		context.Background(),
		7,
		20,
		30,
		40,
	)

	if !errors.Is(err, ErrAssignmentExists) {
		t.Fatalf(
			"expected ErrAssignmentExists, got %v",
			err,
		)
	}

	if database.calls != 4 {
		t.Fatalf("expected 4 database calls, got %d", database.calls)
	}
}
