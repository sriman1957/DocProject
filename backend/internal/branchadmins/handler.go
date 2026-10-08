package branchadmins

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"docproject/backend/internal/auth"
)

type assignRequest struct {
	BranchID int64 `json:"branch_id"`
	UserID   int64 `json:"user_id"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type service interface {
	Assign(
		ctx context.Context,
		collegeID int64,
		branchID int64,
		userID int64,
		assignedBy int64,
	) (Assignment, error)
}

func NewHandler(s service) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /branch-admins", func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{
				Error: "unauthorized",
			})
			return
		}

		// Only College Admins can assign Branch Admins.
		if claims.Role != "college_admin" {
			writeJSON(w, http.StatusForbidden, errorResponse{
				Error: "forbidden",
			})
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

		var request assignRequest

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

		assignment, err := s.Assign(
			r.Context(),
			claims.CollegeID,
			request.BranchID,
			request.UserID,
			claims.UserID,
		)

		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error: "invalid branch admin input",
			})
			return

		case errors.Is(err, ErrBranchNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{
				Error: "branch not found",
			})
			return

		case errors.Is(err, ErrUserNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{
				Error: "eligible faculty user not found",
			})
			return

		case errors.Is(err, ErrAssignmentExists):
			writeJSON(w, http.StatusConflict, errorResponse{
				Error: "user is already a branch admin",
			})
			return

		case err != nil:
			writeJSON(w, http.StatusInternalServerError, errorResponse{
				Error: "internal server error",
			})
			return
		}

		writeJSON(w, http.StatusCreated, assignment)
	})

	return mux
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}
