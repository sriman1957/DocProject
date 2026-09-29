package subgroups

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"docproject/backend/internal/auth"
)

type createSubgroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type subgroupService interface {
	Create(
		ctx context.Context,
		collegeID int64,
		actorID int64,
		actorRole string,
		groupID int64,
		input CreateInput,
	) (Subgroup, error)

	List(
		ctx context.Context,
		collegeID int64,
		actorID int64,
		actorRole string,
		groupID int64,
	) ([]Subgroup, error)
}

// NewHandler creates the HTTP handler for subgroup operations.
func NewHandler(service subgroupService) http.Handler {
	mux := http.NewServeMux()

	// Create a subgroup within a group.
	mux.HandleFunc(
		"POST /groups/{group_id}/subgroups",
		func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.ClaimsFromContext(r.Context())
			if !ok {
				writeJSON(w, http.StatusUnauthorized, errorResponse{
					Error: "unauthorized",
				})
				return
			}

			if claims.Role != "college_admin" &&
				claims.Role != "faculty" {
				writeJSON(w, http.StatusForbidden, errorResponse{
					Error: "forbidden",
				})
				return
			}

			groupID, err := strconv.ParseInt(
				r.PathValue("group_id"),
				10,
				64,
			)
			if err != nil || groupID <= 0 {
				writeJSON(w, http.StatusBadRequest, errorResponse{
					Error: "invalid group ID",
				})
				return
			}

			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

			var request createSubgroupRequest

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

			subgroup, err := service.Create(
				r.Context(),
				claims.CollegeID,
				claims.UserID,
				claims.Role,
				groupID,
				CreateInput{
					Name:        request.Name,
					Description: request.Description,
				},
			)

			switch {
			case errors.Is(err, ErrInvalidInput):
				writeJSON(w, http.StatusBadRequest, errorResponse{
					Error: "invalid subgroup input",
				})
				return

			case errors.Is(err, ErrForbidden):
				writeJSON(w, http.StatusForbidden, errorResponse{
					Error: "forbidden",
				})
				return

			case errors.Is(err, ErrGroupNotFound):
				writeJSON(w, http.StatusNotFound, errorResponse{
					Error: "group not found",
				})
				return

			case errors.Is(err, ErrSubgroupExists):
				writeJSON(w, http.StatusConflict, errorResponse{
					Error: "an active subgroup with this name already exists",
				})
				return

			case err != nil:
				writeJSON(w, http.StatusInternalServerError, errorResponse{
					Error: "internal server error",
				})
				return
			}

			writeJSON(w, http.StatusCreated, subgroup)
		},
	)

	// List subgroups within a group.
	mux.HandleFunc(
		"GET /groups/{group_id}/subgroups",
		func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.ClaimsFromContext(r.Context())
			if !ok {
				writeJSON(w, http.StatusUnauthorized, errorResponse{
					Error: "unauthorized",
				})
				return
			}

			switch claims.Role {
			case "college_admin", "faculty", "student":
				// The service enforces group membership and tenant access.
			default:
				writeJSON(w, http.StatusForbidden, errorResponse{
					Error: "forbidden",
				})
				return
			}

			groupID, err := strconv.ParseInt(
				r.PathValue("group_id"),
				10,
				64,
			)
			if err != nil || groupID <= 0 {
				writeJSON(w, http.StatusBadRequest, errorResponse{
					Error: "invalid group ID",
				})
				return
			}

			subgroups, err := service.List(
				r.Context(),
				claims.CollegeID,
				claims.UserID,
				claims.Role,
				groupID,
			)

			switch {
			case errors.Is(err, ErrInvalidInput):
				writeJSON(w, http.StatusBadRequest, errorResponse{
					Error: "invalid subgroup input",
				})
				return

			case errors.Is(err, ErrForbidden):
				writeJSON(w, http.StatusForbidden, errorResponse{
					Error: "forbidden",
				})
				return

			case errors.Is(err, ErrGroupNotFound):
				writeJSON(w, http.StatusNotFound, errorResponse{
					Error: "group not found",
				})
				return

			case err != nil:
				writeJSON(w, http.StatusInternalServerError, errorResponse{
					Error: "internal server error",
				})
				return
			}

			writeJSON(w, http.StatusOK, subgroups)
		},
	)

	return mux
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}
