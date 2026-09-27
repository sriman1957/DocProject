package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

type fakeQuerier struct {
	row  pgx.Row
	args []any
}

func (f *fakeQuerier) QueryRow(
	_ context.Context,
	_ string,
	args ...any,
) pgx.Row {
	f.args = args
	return f.row
}

type fakeRow struct {
	id           int64
	collegeID    int64
	fullName     string
	email        string
	facultyCode  *string
	role         string
	passwordHash string
	isActive     bool
	err          error
}

func (r *fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}

	*dest[0].(*int64) = r.id
	*dest[1].(*int64) = r.collegeID
	*dest[2].(*string) = r.fullName
	*dest[3].(*string) = r.email
	*dest[4].(**string) = r.facultyCode
	*dest[5].(*string) = r.role
	*dest[6].(*string) = r.passwordHash
	*dest[7].(*bool) = r.isActive

	return nil
}

func TestLoginSuccess(t *testing.T) {
	password := "StrongPassword123!"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	facultyCode := "FAC001"

	database := &fakeQuerier{
		row: &fakeRow{
			id:           1,
			collegeID:    10,
			fullName:     "Test Faculty",
			email:        "faculty@example.com",
			facultyCode:  &facultyCode,
			role:         "faculty",
			passwordHash: hash,
			isActive:     true,
		},
	}

	service := NewService(database)

	user, err := service.Login(
		context.Background(),
		"COLLEGE001",
		"faculty@example.com",
		password,
	)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if user.ID != 1 {
		t.Errorf("user.ID = %d, want 1", user.ID)
	}
	if user.CollegeID != 10 {
		t.Errorf("user.CollegeID = %d, want 10", user.CollegeID)
	}
	if user.FacultyCode != facultyCode {
		t.Errorf("user.FacultyCode = %q, want %q", user.FacultyCode, facultyCode)
	}
	if user.Role != "faculty" {
		t.Errorf("user.Role = %q, want faculty", user.Role)
	}

	if len(database.args) != 2 {
		t.Fatalf("query args count = %d, want 2", len(database.args))
	}
	if database.args[0] != "COLLEGE001" {
		t.Errorf("college code arg = %v, want COLLEGE001", database.args[0])
	}
	if database.args[1] != "faculty@example.com" {
		t.Errorf("email arg = %v, want faculty@example.com", database.args[1])
	}
}

func TestLoginWithNullFacultyCode(t *testing.T) {
	hash, err := HashPassword("StrongPassword123!")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	database := &fakeQuerier{
		row: &fakeRow{
			id:           2,
			collegeID:    10,
			fullName:     "Test Student",
			email:        "student@example.com",
			facultyCode:  nil,
			role:         "student",
			passwordHash: hash,
			isActive:     true,
		},
	}

	service := NewService(database)

	user, err := service.Login(
		context.Background(),
		"COLLEGE001",
		"student@example.com",
		"StrongPassword123!",
	)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if user.FacultyCode != "" {
		t.Errorf("user.FacultyCode = %q, want empty", user.FacultyCode)
	}
}

func TestLoginRejectsInvalidCredentials(t *testing.T) {
	hash, err := HashPassword("CorrectPassword123!")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	tests := []struct {
		name     string
		password string
		active   bool
		rowErr   error
	}{
		{
			name:     "incorrect password",
			password: "WrongPassword123!",
			active:   true,
		},
		{
			name:     "inactive account",
			password: "CorrectPassword123!",
			active:   false,
		},
		{
			name:     "user not found",
			password: "CorrectPassword123!",
			active:   true,
			rowErr:   pgx.ErrNoRows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			database := &fakeQuerier{
				row: &fakeRow{
					id:           1,
					collegeID:    10,
					fullName:     "Test User",
					email:        "user@example.com",
					role:         "student",
					passwordHash: hash,
					isActive:     tt.active,
					err:          tt.rowErr,
				},
			}

			service := NewService(database)

			_, err := service.Login(
				context.Background(),
				"COLLEGE001",
				"user@example.com",
				tt.password,
			)

			if !errors.Is(err, ErrInvalidCredentials) {
				t.Errorf(
					"Login() error = %v, want ErrInvalidCredentials",
					err,
				)
			}
		})
	}
}

func TestLoginRejectsEmptyInput(t *testing.T) {
	tests := []struct {
		name        string
		collegeCode string
		email       string
		password    string
	}{
		{
			name:        "empty college code",
			collegeCode: "",
			email:       "user@example.com",
			password:    "Password123!",
		},
		{
			name:        "empty email",
			collegeCode: "COLLEGE001",
			email:       "",
			password:    "Password123!",
		},
		{
			name:        "empty password",
			collegeCode: "COLLEGE001",
			email:       "user@example.com",
			password:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			database := &fakeQuerier{}
			service := NewService(database)

			_, err := service.Login(
				context.Background(),
				tt.collegeCode,
				tt.email,
				tt.password,
			)

			if !errors.Is(err, ErrInvalidCredentials) {
				t.Errorf(
					"Login() error = %v, want ErrInvalidCredentials",
					err,
				)
			}

			if database.row != nil {
				t.Error("database query should not run for empty input")
			}
		})
	}
}
