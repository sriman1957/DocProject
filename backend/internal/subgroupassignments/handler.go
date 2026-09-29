package subgroupassignments

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"docproject/backend/internal/auth"
)

type createAssignmentRequest struct {
	FacultyID int64 `json:"faculty_id"`
}

type assignmentErrorResponse struct {
	Error string `json:"error"`
}

type assignmentService interface {
	Create(
		ctx context.Context,
		collegeID int64,
		actorID int64,
		actorRole string,
		groupID int64,
		subgroupID int64,
		facultyID int64,
	) (Assignment, error)

	List(
		ctx context.Context,
		collegeID int64,
		actorID int64,
		actorRole string,
		groupID int64,
		subgroupID int64,
	) ([]Assignment, error)

	Revoke(
		ctx context.Context,
		collegeID int64,
		actorID int64,
		actorRole string,
		groupID int64,
		subgroupID int64,
		facultyID int64,
	) (Assignment, error)
}

// NewHandler creates the HTTP handler for subgroup faculty assignments.
func NewHandler(service assignmentService) http.Handler {
	mux := http.NewServeMux()

	// Assign a faculty member to a subgroup.
	mux.HandleFunc(
		"POST /groups/{group_id}/subgroups/{subgroup_id}/faculty",
		func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.ClaimsFromContext(r.Context())
			if !ok {
				writeAssignmentJSON(
					w,
					http.StatusUnauthorized,
					assignmentErrorResponse{Error: "unauthorized"},
				)
				return
			}

			if !canManageAssignments(claims.Role) {
				writeAssignmentJSON(
					w,
					http.StatusForbidden,
					assignmentErrorResponse{Error: "forbidden"},
				)
				return
			}

			groupID, subgroupID, ok := parseAssignmentPathIDs(w, r)
			if !ok {
				return
			}

			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

			var request createAssignmentRequest

			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()

			if err := decoder.Decode(&request); err != nil {
				writeAssignmentJSON(
					w,
					http.StatusBadRequest,
					assignmentErrorResponse{Error: "invalid request body"},
				)
				return
			}

			// Reject trailing JSON values or malformed trailing data.
			var extra any
			if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
				writeAssignmentJSON(
					w,
					http.StatusBadRequest,
					assignmentErrorResponse{Error: "invalid request body"},
				)
				return
			}

			if request.FacultyID <= 0 {
				writeAssignmentJSON(
					w,
					http.StatusBadRequest,
					assignmentErrorResponse{Error: "invalid faculty ID"},
				)
				return
			}

			assignment, err := service.Create(
				r.Context(),
				claims.CollegeID,
				claims.UserID,
				claims.Role,
				groupID,
				subgroupID,
				request.FacultyID,
			)
			if handleAssignmentError(w, err) {
				return
			}

			writeAssignmentJSON(w, http.StatusCreated, assignment)
		},
	)

	// List assignment records, including revoked assignments.
	mux.HandleFunc(
		"GET /groups/{group_id}/subgroups/{subgroup_id}/faculty",
		func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.ClaimsFromContext(r.Context())
			if !ok {
				writeAssignmentJSON(
					w,
					http.StatusUnauthorized,
					assignmentErrorResponse{Error: "unauthorized"},
				)
				return
			}

			if !canManageAssignments(claims.Role) {
				writeAssignmentJSON(
					w,
					http.StatusForbidden,
					assignmentErrorResponse{Error: "forbidden"},
				)
				return
			}

			groupID, subgroupID, ok := parseAssignmentPathIDs(w, r)
			if !ok {
				return
			}

			assignments, err := service.List(
				r.Context(),
				claims.CollegeID,
				claims.UserID,
				claims.Role,
				groupID,
				subgroupID,
			)
			if handleAssignmentError(w, err) {
				return
			}

			writeAssignmentJSON(w, http.StatusOK, assignments)
		},
	)

	// Revoke an active assignment without deleting its history.
	mux.HandleFunc(
		"DELETE /groups/{group_id}/subgroups/{subgroup_id}/faculty/{faculty_id}",
		func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.ClaimsFromContext(r.Context())
			if !ok {
				writeAssignmentJSON(
					w,
					http.StatusUnauthorized,
					assignmentErrorResponse{Error: "unauthorized"},
				)
				return
			}

			if !canManageAssignments(claims.Role) {
				writeAssignmentJSON(
					w,
					http.StatusForbidden,
					assignmentErrorResponse{Error: "forbidden"},
				)
				return
			}

			groupID, subgroupID, ok := parseAssignmentPathIDs(w, r)
			if !ok {
				return
			}

			facultyID, err := strconv.ParseInt(
				r.PathValue("faculty_id"),
				10,
				64,
			)
			if err != nil || facultyID <= 0 {
				writeAssignmentJSON(
					w,
					http.StatusBadRequest,
					assignmentErrorResponse{Error: "invalid faculty ID"},
				)
				return
			}

			assignment, err := service.Revoke(
				r.Context(),
				claims.CollegeID,
				claims.UserID,
				claims.Role,
				groupID,
				subgroupID,
				facultyID,
			)
			if handleAssignmentError(w, err) {
				return
			}

			writeAssignmentJSON(w, http.StatusOK, assignment)
		},
	)

	return mux
}

// canManageAssignments performs the handler-level role check.
// The service additionally verifies group membership and college scope.
func canManageAssignments(role string) bool {
	return role == "college_admin" || role == "faculty"
}

// parseAssignmentPathIDs validates the group and subgroup path parameters.
func parseAssignmentPathIDs(
	w http.ResponseWriter,
	r *http.Request,
) (int64, int64, bool) {
	groupID, err := strconv.ParseInt(
		r.PathValue("group_id"),
		10,
		64,
	)
	if err != nil || groupID <= 0 {
		writeAssignmentJSON(
			w,
			http.StatusBadRequest,
			assignmentErrorResponse{Error: "invalid group ID"},
		)
		return 0, 0, false
	}

	subgroupID, err := strconv.ParseInt(
		r.PathValue("subgroup_id"),
		10,
		64,
	)
	if err != nil || subgroupID <= 0 {
		writeAssignmentJSON(
			w,
			http.StatusBadRequest,
			assignmentErrorResponse{Error: "invalid subgroup ID"},
		)
		return 0, 0, false
	}

	return groupID, subgroupID, true
}

// handleAssignmentError maps service errors to HTTP responses.
// It returns true when an error was handled.
func handleAssignmentError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Is(err, ErrInvalidInput):
		writeAssignmentJSON(
			w,
			http.StatusBadRequest,
			assignmentErrorResponse{Error: "invalid assignment input"},
		)

	case errors.Is(err, ErrForbidden):
		writeAssignmentJSON(
			w,
			http.StatusForbidden,
			assignmentErrorResponse{Error: "forbidden"},
		)

	case errors.Is(err, ErrGroupNotFound):
		writeAssignmentJSON(
			w,
			http.StatusNotFound,
			assignmentErrorResponse{Error: "group not found"},
		)

	case errors.Is(err, ErrSubgroupNotFound):
		writeAssignmentJSON(
			w,
			http.StatusNotFound,
			assignmentErrorResponse{Error: "subgroup not found"},
		)

	case errors.Is(err, ErrFacultyNotEligible):
		writeAssignmentJSON(
			w,
			http.StatusUnprocessableEntity,
			assignmentErrorResponse{
				Error: "faculty is not eligible for this subgroup",
			},
		)

	case errors.Is(err, ErrAssignmentExists):
		writeAssignmentJSON(
			w,
			http.StatusConflict,
			assignmentErrorResponse{Error: "active assignment already exists"},
		)

	case errors.Is(err, ErrAssignmentNotFound):
		writeAssignmentJSON(
			w,
			http.StatusNotFound,
			assignmentErrorResponse{Error: "active assignment not found"},
		)

	default:
		writeAssignmentJSON(
			w,
			http.StatusInternalServerError,
			assignmentErrorResponse{Error: "internal server error"},
		)
	}

	return true
}

func writeAssignmentJSON(
	w http.ResponseWriter,
	status int,
	data any,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}
