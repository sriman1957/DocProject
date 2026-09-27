package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const claimsContextKey contextKey = "auth_claims"

// AuthMiddleware validates access tokens before allowing a request
// to reach a protected handler.
func AuthMiddleware(
	tokens *TokenService,
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const bearerPrefix = "Bearer "

		authHeader := r.Header.Get("Authorization")

		if !strings.HasPrefix(authHeader, bearerPrefix) {
			writeError(
				w,
				http.StatusUnauthorized,
				"missing or invalid authorization header",
			)
			return
		}

		tokenString := strings.TrimSpace(
			strings.TrimPrefix(authHeader, bearerPrefix),
		)

		if tokenString == "" || strings.ContainsAny(tokenString, " \t\r\n") {
			writeError(
				w,
				http.StatusUnauthorized,
				"missing or invalid authorization header",
			)
			return
		}

		claims, err := tokens.Validate(tokenString)
		if err != nil {
			writeError(
				w,
				http.StatusUnauthorized,
				"invalid or expired access token",
			)
			return
		}

		if claims.UserID <= 0 ||
			claims.CollegeID <= 0 ||
			!isValidRole(claims.Role) {
			writeError(
				w,
				http.StatusUnauthorized,
				"invalid access token claims",
			)
			return
		}

		ctx := context.WithValue(
			r.Context(),
			claimsContextKey,
			claims,
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ClaimsFromContext retrieves authenticated claims from a request context.
func ClaimsFromContext(ctx context.Context) (*AccessClaims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*AccessClaims)
	return claims, ok
}

func isValidRole(role string) bool {
	switch role {
	case "student", "faculty", "college_admin":
		return true
	default:
		return false
	}
}
