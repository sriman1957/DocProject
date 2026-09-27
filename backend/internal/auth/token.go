package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid access token")
	ErrTokenExpired = errors.New("access token expired")
)

// AccessClaims contains the identity and authorization claims
// embedded in an access token.
type AccessClaims struct {
	UserID    int64  `json:"user_id"`
	CollegeID int64  `json:"college_id"`
	Role      string `json:"role"`

	jwt.RegisteredClaims
}

// TokenService creates and validates JWT access tokens.
type TokenService struct {
	secret []byte
	ttl    time.Duration
}

// NewTokenService creates a JWT token service.
func NewTokenService(secret string, ttl time.Duration) *TokenService {
	return &TokenService{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

// Generate creates a signed access token for an authenticated user.
func (s *TokenService) Generate(user User) (string, error) {
	if len(s.secret) < 32 {
		return "", errors.New("JWT secret must be at least 32 bytes")
	}

	if s.ttl <= 0 {
		return "", errors.New("JWT access token TTL must be positive")
	}

	now := time.Now()

	claims := AccessClaims{
		UserID:    user.ID,
		CollegeID: user.CollegeID,
		Role:      user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", user.ID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}

	return signedToken, nil
}

// Validate verifies the signature, signing algorithm, and expiry
// of an access token.
func (s *TokenService) Validate(tokenString string) (*AccessClaims, error) {
	if len(s.secret) < 32 {
		return nil, errors.New("JWT secret must be at least 32 bytes")
	}

	claims := &AccessClaims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, ErrInvalidToken
			}

			return s.secret, nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}

		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
