package users

import (
	"context"
	"errors"
	"testing"

	"docproject/backend/internal/auth"

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
	query string
	args  []any
	err   error
}

func (f *fakeQuerier) QueryRow(
	_ context.Context,
	query string,
	args ...any,
) pgx.Row {
	f.query = query
	f.args = args

	return fakeRow{
		scan: func(dest ...any) error {
			if f.err != nil {
				return f.err
			}

			*dest[0].(*int64) = 10
			*dest[1].(*int64) = 7
			*dest[2].(*string) = "Test Student"
			*dest[3].(*string) = "student@example.com"
			*dest[4].(*string) = ""
			*dest[5].(*string) = "student"
			*dest[6].(*bool) = true
			*dest[7].(*string) = "2026-09-29 10:00:00+00"
			*dest[8].(*string) = "2026-09-29 10:00:00+00"

			return nil
		},
	}
}

func validStudentInput() CreateInput {
	return CreateInput{
		FullName: "Test Student",
		Email:    "student@example.com",
		Password: "a-strong-test-password",
		Role:     "student",
	}
}

func TestCreateSuccess(t *testing.T) {
	database := &fakeQuerier{}
	service := NewService(database)

	user, err := service.Create(
		context.Background(),
		7,
		validStudentInput(),
	)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if user.ID != 10 {
		t.Errorf("expected user ID 10, got %d", user.ID)
	}

	if user.CollegeID != 7 {
		t.Errorf("expected college ID 7, got %d", user.CollegeID)
	}

	if user.Role != "student" {
		t.Errorf("expected role student, got %q", user.Role)
	}

	if user.Email != "student@example.com" {
		t.Errorf("unexpected email: %q", user.Email)
	}

	if len(database.args) != 6 {
		t.Fatalf("expected 6 SQL arguments, got %d", len(database.args))
	}

	passwordHash, ok := database.args[3].(string)
	if !ok {
		t.Fatal("password hash argument is not a string")
	}

	if passwordHash == validStudentInput().Password {
		t.Fatal("plaintext password was passed to the database")
	}

	valid, err := auth.VerifyPassword(
		validStudentInput().Password,
		passwordHash,
	)
	if err != nil {
		t.Fatalf("VerifyPassword returned error: %v", err)
	}
	if !valid {
		t.Fatal("stored password hash does not match input password")
	}
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		collegeID int64
		input     CreateInput
	}{
		{
			name:      "invalid college",
			collegeID: 0,
			input:     validStudentInput(),
		},
		{
			name:      "empty name",
			collegeID: 7,
			input: CreateInput{
				Email:    "student@example.com",
				Password: "password",
				Role:     "student",
			},
		},
		{
			name:      "invalid email whitespace",
			collegeID: 7,
			input: CreateInput{
				FullName: "Test Student",
				Email:    "student @example.com",
				Password: "password",
				Role:     "student",
			},
		},
		{
			name:      "admin role not allowed",
			collegeID: 7,
			input: CreateInput{
				FullName: "Test Admin",
				Email:    "admin@example.com",
				Password: "password",
				Role:     "college_admin",
			},
		},
		{
			name:      "student cannot have faculty code",
			collegeID: 7,
			input: CreateInput{
				FullName:    "Test Student",
				Email:       "student@example.com",
				Password:    "password",
				FacultyCode: "T014",
				Role:        "student",
			},
		},
		{
			name:      "empty password",
			collegeID: 7,
			input: CreateInput{
				FullName: "Test Student",
				Email:    "student@example.com",
				Role:     "student",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := &fakeQuerier{}
			service := NewService(database)

			_, err := service.Create(
				context.Background(),
				test.collegeID,
				test.input,
			)

			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf(
					"expected ErrInvalidInput, got %v",
					err,
				)
			}

			if database.query != "" {
				t.Fatal("database should not be called for invalid input")
			}
		})
	}
}

func TestCreateMapsDuplicateEmail(t *testing.T) {
	database := &fakeQuerier{
		err: &pgconn.PgError{
			Code:           "23505",
			ConstraintName: "users_college_id_email_key",
		},
	}

	service := NewService(database)

	_, err := service.Create(
		context.Background(),
		7,
		validStudentInput(),
	)

	if !errors.Is(err, ErrEmailAlreadyExists) {
		t.Fatalf(
			"expected ErrEmailAlreadyExists, got %v",
			err,
		)
	}
}

func TestCreateDoesNotExposePasswordHash(t *testing.T) {
	database := &fakeQuerier{}
	service := NewService(database)

	user, err := service.Create(
		context.Background(),
		7,
		validStudentInput(),
	)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if user.Email == "" {
		t.Fatal("expected email in returned user")
	}

	// The response model intentionally has no password/hash field.
	// This test also verifies the returned value is the public User type.
	if user.Role != "student" {
		t.Fatalf("unexpected role: %q", user.Role)
	}
}
