package users

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"docproject/backend/internal/auth"
)

type createUserRequest struct {
	FullName    string `json:"full_name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	FacultyCode string `json:"faculty_code"`
	Role        string `json:"role"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type userService interface {
	Create(
		ctx context.Context,
		collegeID int64,
		input CreateInput,
	) (User, error)
}

// NewHandler creates the HTTP handler for user operations.
func NewHandler(service userService) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /users", func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{
				Error: "unauthorized",
			})
			return
		}

		if claims.Role != "college_admin" {
			writeJSON(w, http.StatusForbidden, errorResponse{
				Error: "forbidden",
			})
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

		var request createUserRequest

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error: "invalid request body",
			})
			return
		}

		var extra any
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error: "invalid request body",
			})
			return
		}

		user, err := service.Create(
			r.Context(),
			claims.CollegeID,
			CreateInput{
				FullName:    request.FullName,
				Email:       request.Email,
				Password:    request.Password,
				FacultyCode: request.FacultyCode,
				Role:        request.Role,
			},
		)

		if errors.Is(err, ErrInvalidInput) {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error: "invalid user input",
			})
			return
		}

		if errors.Is(err, ErrEmailAlreadyExists) {
			writeJSON(w, http.StatusConflict, errorResponse{
				Error: "email already exists in this college",
			})
			return
		}

		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{
				Error: "internal server error",
			})
			return
		}

		writeJSON(w, http.StatusCreated, user)
	})

	return mux
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}
