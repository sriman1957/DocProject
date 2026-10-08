package groups

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"docproject/backend/internal/auth"
)

type createGroupRequest struct {
	BranchID    int64  `json:"branch_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type addMemberRequest struct {
	UserID int64 `json:"user_id"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type groupService interface {
	Create(
		ctx context.Context,
		collegeID int64,
		createdBy int64,
		role string,
		input CreateInput,
	) (Group, error)

	List(
		ctx context.Context,
		collegeID int64,
		userID int64,
		role string,
	) ([]Group, error)

	AddMember(
		ctx context.Context,
		collegeID int64,
		userID int64,
		role string,
		groupID int64,
		input AddMemberInput,
	) (Member, error)

	ListMembers(
		ctx context.Context,
		collegeID int64,
		userID int64,
		role string,
		groupID int64,
	) ([]Member, error)
}

// NewHandler creates the HTTP handler for group operations.
func NewHandler(service groupService) http.Handler {
	mux := http.NewServeMux()

	// Create a group / batch.
	mux.HandleFunc("POST /groups", func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{
				Error: "unauthorized",
			})
			return
		}

		// College admins and faculty users can reach the service.
		//
		// For faculty, the service determines whether the user is actually
		// an active Branch Admin. Ordinary faculty users will receive 403
		// from the service.
		if claims.Role != "college_admin" && claims.Role != "faculty" {
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
			claims.Role,
			CreateInput{
				BranchID:    request.BranchID,
				Name:        request.Name,
				Description: request.Description,
			},
		)

		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error: "invalid group input",
			})
			return

		case errors.Is(err, ErrBranchNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{
				Error: "branch not found",
			})
			return

		case errors.Is(err, ErrForbidden):
			writeJSON(w, http.StatusForbidden, errorResponse{
				Error: "forbidden",
			})
			return

		case err != nil:
			writeJSON(w, http.StatusInternalServerError, errorResponse{
				Error: "internal server error",
			})
			return
		}

		writeJSON(w, http.StatusCreated, group)
	})

	// List groups / batches visible to the caller.
	mux.HandleFunc("GET /groups", func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{
				Error: "unauthorized",
			})
			return
		}

		if claims.Role != "college_admin" && claims.Role != "faculty" {
			writeJSON(w, http.StatusForbidden, errorResponse{
				Error: "forbidden",
			})
			return
		}

		groups, err := service.List(
			r.Context(),
			claims.CollegeID,
			claims.UserID,
			claims.Role,
		)

		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error: "invalid group input",
			})
			return

		case errors.Is(err, ErrForbidden):
			writeJSON(w, http.StatusForbidden, errorResponse{
				Error: "forbidden",
			})
			return

		case err != nil:
			writeJSON(w, http.StatusInternalServerError, errorResponse{
				Error: "internal server error",
			})
			return
		}

		writeJSON(w, http.StatusOK, groups)
	})

	// List members of a group.
	mux.HandleFunc("GET /groups/{group_id}/members", func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{
				Error: "unauthorized",
			})
			return
		}

		if claims.Role != "college_admin" && claims.Role != "faculty" {
			writeJSON(w, http.StatusForbidden, errorResponse{
				Error: "forbidden",
			})
			return
		}

		groupID, err := strconv.ParseInt(r.PathValue("group_id"), 10, 64)
		if err != nil || groupID <= 0 {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error: "invalid group ID",
			})
			return
		}

		members, err := service.ListMembers(
			r.Context(),
			claims.CollegeID,
			claims.UserID,
			claims.Role,
			groupID,
		)

		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error: "invalid group input",
			})
			return

		case errors.Is(err, ErrGroupNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{
				Error: "group not found",
			})
			return

		case errors.Is(err, ErrForbidden):
			writeJSON(w, http.StatusForbidden, errorResponse{
				Error: "forbidden",
			})
			return

		case err != nil:
			writeJSON(w, http.StatusInternalServerError, errorResponse{
				Error: "internal server error",
			})
			return
		}

		writeJSON(w, http.StatusOK, members)
	})

	// Add a member to a group.
	mux.HandleFunc("POST /groups/{group_id}/members", func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{
				Error: "unauthorized",
			})
			return
		}

		if claims.Role != "college_admin" && claims.Role != "faculty" {
			writeJSON(w, http.StatusForbidden, errorResponse{
				Error: "forbidden",
			})
			return
		}

		groupID, err := strconv.ParseInt(r.PathValue("group_id"), 10, 64)
		if err != nil || groupID <= 0 {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error: "invalid group ID",
			})
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

		var request addMemberRequest

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

		member, err := service.AddMember(
			r.Context(),
			claims.CollegeID,
			claims.UserID,
			claims.Role,
			groupID,
			AddMemberInput{
				UserID: request.UserID,
			},
		)

		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error: "invalid membership input",
			})
			return

		case errors.Is(err, ErrGroupNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{
				Error: "group not found",
			})
			return

		case errors.Is(err, ErrMemberNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{
				Error: "eligible user not found",
			})
			return

		case errors.Is(err, ErrMembershipExists):
			writeJSON(w, http.StatusConflict, errorResponse{
				Error: "user is already a member of this group",
			})
			return

		case errors.Is(err, ErrForbidden):
			writeJSON(w, http.StatusForbidden, errorResponse{
				Error: "forbidden",
			})
			return

		case err != nil:
			writeJSON(w, http.StatusInternalServerError, errorResponse{
				Error: "internal server error",
			})
			return
		}

		writeJSON(w, http.StatusCreated, member)
	})

	return mux
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}
