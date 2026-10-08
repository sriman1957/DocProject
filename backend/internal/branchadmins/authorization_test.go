package branchadmins

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

type authorizationFakeRow struct {
	scan func(dest ...any) error
}

func (r authorizationFakeRow) Scan(dest ...any) error {
	return r.scan(dest...)
}

type authorizationFakeQuerier struct {
	query string
	args  []any
	err   error
	value bool
}

func (f *authorizationFakeQuerier) QueryRow(
	_ context.Context,
	query string,
	args ...any,
) pgx.Row {
	f.query = query
	f.args = args

	return authorizationFakeRow{
		scan: func(dest ...any) error {
			if f.err != nil {
				return f.err
			}

			if len(dest) != 1 {
				return errors.New(
					"expected one scan destination",
				)
			}

			target, ok := dest[0].(*bool)
			if !ok {
				return errors.New(
					"expected *bool destination",
				)
			}

			*target = f.value

			return nil
		},
	}
}

func TestIsBranchAdminSuccess(t *testing.T) {
	db := &authorizationFakeQuerier{
		value: true,
	}

	ok, err := IsBranchAdmin(
		context.Background(),
		db,
		7,
		10,
		3,
	)
	if err != nil {
		t.Fatalf(
			"IsBranchAdmin returned error: %v",
			err,
		)
	}

	if !ok {
		t.Fatal("expected user to be a branch admin")
	}

	if len(db.args) != 3 {
		t.Fatalf(
			"expected 3 query arguments, got %d",
			len(db.args),
		)
	}

	if db.args[0] != int64(7) {
		t.Errorf(
			"expected college ID 7, got %v",
			db.args[0],
		)
	}

	if db.args[1] != int64(3) {
		t.Errorf(
			"expected branch ID 3, got %v",
			db.args[1],
		)
	}

	if db.args[2] != int64(10) {
		t.Errorf(
			"expected user ID 10, got %v",
			db.args[2],
		)
	}
}

func TestIsBranchAdminNotAssigned(t *testing.T) {
	db := &authorizationFakeQuerier{
		value: false,
	}

	ok, err := IsBranchAdmin(
		context.Background(),
		db,
		7,
		10,
		3,
	)
	if err != nil {
		t.Fatalf(
			"IsBranchAdmin returned error: %v",
			err,
		)
	}

	if ok {
		t.Fatal(
			"expected user not to be a branch admin",
		)
	}
}

func TestIsBranchAdminInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		collegeID int64
		userID    int64
		branchID  int64
	}{
		{
			name:      "invalid college ID",
			collegeID: 0,
			userID:    10,
			branchID:  3,
		},
		{
			name:      "invalid user ID",
			collegeID: 7,
			userID:    0,
			branchID:  3,
		},
		{
			name:      "invalid branch ID",
			collegeID: 7,
			userID:    10,
			branchID:  0,
		},
		{
			name:      "negative college ID",
			collegeID: -1,
			userID:    10,
			branchID:  3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := &authorizationFakeQuerier{}

			ok, err := IsBranchAdmin(
				context.Background(),
				db,
				test.collegeID,
				test.userID,
				test.branchID,
			)
			if err != nil {
				t.Fatalf(
					"IsBranchAdmin returned error: %v",
					err,
				)
			}

			if ok {
				t.Fatal(
					"expected invalid input to return false",
				)
			}

			if db.query != "" {
				t.Fatal(
					"database should not be queried for invalid input",
				)
			}
		})
	}
}

func TestIsBranchAdminDatabaseError(t *testing.T) {
	expectedErr := errors.New("database failure")

	db := &authorizationFakeQuerier{
		err: expectedErr,
	}

	ok, err := IsBranchAdmin(
		context.Background(),
		db,
		7,
		10,
		3,
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected database error, got %v",
			err,
		)
	}

	if ok {
		t.Fatal(
			"expected false when database query fails",
		)
	}
}
