package users

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"docproject/backend/internal/auth"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrInvalidInput       = errors.New("invalid user input")
	ErrEmailAlreadyExists = errors.New("email already exists in this college")
)

type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Service struct {
	db Querier
}

func NewService(database Querier) *Service {
	return &Service{
		db: database,
	}
}

type User struct {
	ID          int64  `json:"id"`
	CollegeID   int64  `json:"college_id"`
	FullName    string `json:"full_name"`
	Email       string `json:"email"`
	FacultyCode string `json:"faculty_code,omitempty"`
	Role        string `json:"role"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type CreateInput struct {
	FullName    string
	Email       string
	Password    string
	FacultyCode string
	Role        string
}

func (s *Service) Create(
	ctx context.Context,
	collegeID int64,
	input CreateInput,
) (User, error) {
	if collegeID <= 0 {
		return User{}, ErrInvalidInput
	}

	fullName := strings.TrimSpace(input.FullName)
	email := strings.TrimSpace(input.Email)
	facultyCode := strings.TrimSpace(input.FacultyCode)

	if fullName == "" || len(fullName) > 200 {
		return User{}, ErrInvalidInput
	}

	if email == "" || len(email) > 254 ||
		strings.ContainsAny(email, " \t\r\n") {
		return User{}, ErrInvalidInput
	}

	if input.Role != "student" && input.Role != "faculty" {
		return User{}, ErrInvalidInput
	}

	if len(facultyCode) > 50 {
		return User{}, ErrInvalidInput
	}

	if input.Role == "student" && facultyCode != "" {
		return User{}, ErrInvalidInput
	}

	if input.Password == "" || len(input.Password) > 1024 {
		return User{}, ErrInvalidInput
	}

	passwordHash, err := auth.HashPassword(input.Password)
	if err != nil {
		return User{}, fmt.Errorf("hash user password: %w", err)
	}

	const query = `
		INSERT INTO users (
			college_id,
			full_name,
			email,
			password_hash,
			faculty_code,
			role
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			NULLIF($5, ''),
			$6
		)
		RETURNING
			id,
			college_id,
			full_name,
			email,
			COALESCE(faculty_code, ''),
			role,
			is_active,
			created_at::text,
			updated_at::text
	`

	var user User

	err = s.db.QueryRow(
		ctx,
		query,
		collegeID,
		fullName,
		email,
		passwordHash,
		facultyCode,
		input.Role,
	).Scan(
		&user.ID,
		&user.CollegeID,
		&user.FullName,
		&user.Email,
		&user.FacultyCode,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) &&
			pgErr.Code == "23505" &&
			pgErr.ConstraintName == "users_college_id_email_key" {
			return User{}, ErrEmailAlreadyExists
		}

		return User{}, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}
