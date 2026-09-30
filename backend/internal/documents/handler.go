package documents

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"docproject/backend/internal/auth"
)

type ServiceInterface interface {
	ListPersonalVault(
		ctx context.Context,
		collegeID int64,
		studentID int64,
	) ([]Document, error)
}

type Handler struct {
	service ServiceInterface
}

func NewHandler(service ServiceInterface) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		http.Error(
			w,
			"unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	if claims.Role != "student" {
		http.Error(
			w,
			"forbidden",
			http.StatusForbidden,
		)
		return
	}

	documents, err := h.service.ListPersonalVault(
		r.Context(),
		claims.CollegeID,
		claims.UserID,
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			http.Error(
				w,
				"invalid input",
				http.StatusBadRequest,
			)

		case errors.Is(err, ErrForbidden):
			http.Error(
				w,
				"forbidden",
				http.StatusForbidden,
			)

		default:
			http.Error(
				w,
				"internal server error",
				http.StatusInternalServerError,
			)
		}

		return
	}

	writeJSON(w, http.StatusOK, documents)
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	value any,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		return
	}
}
