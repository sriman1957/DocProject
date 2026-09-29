package groups

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"docproject/backend/internal/auth"
)

type createGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type groupService interface {
	Create(
		ctx context.Context,
		collegeID int64,
		createdBy int64,
		input CreateInput,
	) (Group, error)

	List(
		ctx context.Context,
		collegeID int64,
	) ([]Group, error)
}

// NewHandler creates the HTTP handler for group operations.
func NewHandler(service groupService) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /groups", func(w http.ResponseWriter, r *http.Request) {
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

		var request createGroupRequest

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

		group, err := service.Create(
			r.Context(),
			claims.CollegeID,
			claims.UserID,
			CreateInput{
				Name:        request.Name,
				Description: request.Description,
			},
		)

		if errors.Is(err, ErrInvalidInput) {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error: "invalid group input",
			})
			return
		}

		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{
				Error: "internal server error",
			})
			return
		}

		writeJSON(w, http.StatusCreated, group)
	})

	mux.HandleFunc("GET /groups", func(w http.ResponseWriter, r *http.Request) {
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

		groups, err := service.List(
			r.Context(),
			claims.CollegeID,
		)
		if errors.Is(err, ErrInvalidInput) {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error: "invalid group input",
			})
			return
		}

		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{
				Error: "internal server error",
			})
			return
		}

		writeJSON(w, http.StatusOK, groups)
	})

	return mux
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}
