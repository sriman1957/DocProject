package accessperiods

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"docproject/backend/internal/auth"
)

type createRequest struct {
	StartsAt string `json:"starts_at"`
	EndsAt   string `json:"ends_at"`
}

type service interface {
	Create(context.Context, int64, int64, string, int64, int64, CreateInput) (AccessPeriod, error)
	List(context.Context, int64, int64, string, int64, int64) ([]AccessPeriod, error)
}

func NewHandler(s service) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /groups/{group_id}/subgroups/{subgroup_id}/access-periods", func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if claims.Role != "college_admin" && claims.Role != "faculty" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}

		groupID, subgroupID, ok := parseIDs(r)
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid ID"})
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		var req createRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		var extra any
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "starts_at must be RFC3339"})
			return
		}
		endsAt, err := time.Parse(time.RFC3339, req.EndsAt)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ends_at must be RFC3339"})
			return
		}

		period, err := s.Create(r.Context(), claims.CollegeID, claims.UserID, claims.Role, groupID, subgroupID, CreateInput{StartsAt: startsAt, EndsAt: endsAt})
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid access period"})
		case errors.Is(err, ErrForbidden):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		case errors.Is(err, ErrGroupNotFound), errors.Is(err, ErrSubgroupNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "group or subgroup not found"})
		case errors.Is(err, ErrPeriodOverlap):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "access period overlaps an existing period"})
		case err != nil:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		default:
			writeJSON(w, http.StatusCreated, period)
		}
	})

	mux.HandleFunc("GET /groups/{group_id}/subgroups/{subgroup_id}/access-periods", func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if claims.Role != "college_admin" && claims.Role != "faculty" && claims.Role != "student" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}

		groupID, subgroupID, ok := parseIDs(r)
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid ID"})
			return
		}

		periods, err := s.List(r.Context(), claims.CollegeID, claims.UserID, claims.Role, groupID, subgroupID)
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid access period"})
		case errors.Is(err, ErrForbidden):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		case errors.Is(err, ErrGroupNotFound), errors.Is(err, ErrSubgroupNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "group or subgroup not found"})
		case err != nil:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		default:
			writeJSON(w, http.StatusOK, periods)
		}
	})

	return mux
}

func parseIDs(r *http.Request) (int64, int64, bool) {
	groupID, err1 := strconv.ParseInt(r.PathValue("group_id"), 10, 64)
	subgroupID, err2 := strconv.ParseInt(r.PathValue("subgroup_id"), 10, 64)
	return groupID, subgroupID, err1 == nil && err2 == nil && groupID > 0 && subgroupID > 0
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
