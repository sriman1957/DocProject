package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

// Querier represents the database operation required by the service.
// *db.Pool satisfies this interface.
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// User contains the public information returned after login.
type User struct {
	ID          int64  `json:"id"`
	CollegeID   int64  `json:"college_id"`
	FullName    string `json:"full_name"`
	Email       string `json:"email"`
	FacultyCode string `json:"faculty_code,omitempty"`
	Role        string `json:"role"`
}

// Service handles authentication operations.
type Service struct {
	db Querier
}

// NewService creates an authentication service.
func NewService(database Querier) *Service {
	return &Service{
		db: database,
	}
}

// Login authenticates a user within a specific college.
func (s *Service) Login(
	ctx context.Context,
	collegeCode string,
	email string,
	password string,
) (User, error) {
	var user User
	var passwordHash string
	var isActive bool
	var facultyCode *string

	collegeCode = strings.TrimSpace(collegeCode)
	email = strings.TrimSpace(email)

	if collegeCode == "" || email == "" || password == "" {
		return User{}, ErrInvalidCredentials
	}

	const query = `
		SELECT
			u.id,
			u.college_id,
			u.full_name,
			u.email,
			u.faculty_code,
			u.role,
			u.password_hash,
			u.is_active
		FROM users AS u
		INNER JOIN colleges AS c
			ON c.id = u.college_id
		WHERE c.code = $1
			AND u.email = $2
	`

	err := s.db.QueryRow(
		ctx,
		query,
		collegeCode,
		email,
	).Scan(
		&user.ID,
		&user.CollegeID,
		&user.FullName,
		&user.Email,
		&facultyCode,
		&user.Role,
		&passwordHash,
		&isActive,
	)

	if facultyCode != nil {
		user.FacultyCode = *facultyCode
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrInvalidCredentials
	}

	if err != nil {
		return User{}, fmt.Errorf("query user for login: %w", err)
	}

	passwordMatches, err := VerifyPassword(password, passwordHash)
	if err != nil {
		return User{}, fmt.Errorf("verify stored password hash: %w", err)
	}

	if !passwordMatches || !isActive {
		return User{}, ErrInvalidCredentials
	}

	return user, nil
}
