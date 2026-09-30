package accessperiods

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"docproject/backend/internal/auth"
)

type createAccessPeriodRequest struct {
	StartsAt string `json:"starts_at"`
	EndsAt   string `json:"ends_at"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type accessPeriodService interface {
	Create(
		ctx context.Context,
		collegeID int64,
		actorID int64,
		actorRole string,
		groupID int64,
		subgroupID int64,
		input CreateInput,
	) (AccessPeriod, error)

	List(
		ctx context.Context,
		collegeID int64,
		actorID int64,
		actorRole string,
		groupID int64,
		subgroupID int64,
	) ([]AccessPeriod, error)
}

// NewHandler creates the HTTP handler for access-period operations.
func NewHandler(service accessPeriodService) http.Handler {
	mux := http.NewServeMux()

	// Create an access period for a subgroup.
	mux.HandleFunc(
		"POST /groups/{group_id}/subgroups/{subgroup_id}/access-periods",
		func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.ClaimsFromContext(r.Context())
			if !ok {
				writeJSON(w, http.StatusUnauthorized, errorResponse{
					Error: "unauthorized",
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

			subgroupID, err := strconv.ParseInt(
				r.PathValue("subgroup_id"),
				10,
				64,
			)
			if err != nil || subgroupID <= 0 {
				writeJSON(w, http.StatusBadRequest, errorResponse{
					Error: "invalid subgroup ID",
				})
				return
			}

			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

			var request createAccessPeriodRequest

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

			period, err := service.Create(
				r.Context(),
				claims.CollegeID,
				claims.UserID,
				claims.Role,
				groupID,
				subgroupID,
				CreateInput{
					StartsAt: request.StartsAt,
					EndsAt:   request.EndsAt,
				},
			)

			switch {
			case errors.Is(err, ErrInvalidInput):
				writeJSON(w, http.StatusBadRequest, errorResponse{
					Error: "invalid access period input",
				})
				return

			case errors.Is(err, ErrForbidden):
				writeJSON(w, http.StatusForbidden, errorResponse{
					Error: "forbidden",
				})
				return

			case errors.Is(err, ErrSubgroupNotFound):
				writeJSON(w, http.StatusNotFound, errorResponse{
					Error: "subgroup not found",
				})
				return

			case errors.Is(err, ErrAccessPeriodOverlap):
				writeJSON(w, http.StatusConflict, errorResponse{
					Error: "access period overlaps an existing period",
				})
				return

			case err != nil:
				writeJSON(w, http.StatusInternalServerError, errorResponse{
					Error: "internal server error",
				})
				return
			}

			writeJSON(w, http.StatusCreated, period)
		},
	)

	// List access periods for a subgroup.
	mux.HandleFunc(
		"GET /groups/{group_id}/subgroups/{subgroup_id}/access-periods",
		func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.ClaimsFromContext(r.Context())
			if !ok {
				writeJSON(w, http.StatusUnauthorized, errorResponse{
					Error: "unauthorized",
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

			subgroupID, err := strconv.ParseInt(
				r.PathValue("subgroup_id"),
				10,
				64,
			)
			if err != nil || subgroupID <= 0 {
				writeJSON(w, http.StatusBadRequest, errorResponse{
					Error: "invalid subgroup ID",
				})
				return
			}

			periods, err := service.List(
				r.Context(),
				claims.CollegeID,
				claims.UserID,
				claims.Role,
				groupID,
				subgroupID,
			)

			switch {
			case errors.Is(err, ErrInvalidInput):
				writeJSON(w, http.StatusBadRequest, errorResponse{
					Error: "invalid access period input",
				})
				return

			case errors.Is(err, ErrForbidden):
				writeJSON(w, http.StatusForbidden, errorResponse{
					Error: "forbidden",
				})
				return

			case errors.Is(err, ErrSubgroupNotFound):
				writeJSON(w, http.StatusNotFound, errorResponse{
					Error: "subgroup not found",
				})
				return

			case err != nil:
				writeJSON(w, http.StatusInternalServerError, errorResponse{
					Error: "internal server error",
				})
				return
			}

			writeJSON(w, http.StatusOK, periods)
		},
	)

	return mux
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}
