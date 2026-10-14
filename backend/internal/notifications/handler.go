package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"docproject/backend/internal/auth"
)

type notificationService interface {
	List(
		ctx context.Context,
		collegeID int64,
		userID int64,
	) ([]Notification, error)

	MarkRead(
		ctx context.Context,
		collegeID int64,
		userID int64,
		notificationID int64,
	) (Notification, error)

	UnreadCount(
		ctx context.Context,
		collegeID int64,
		userID int64,
	) (int64, error)
}

// NewHandler creates the HTTP handler for notification operations.
func NewHandler(service notificationService) http.Handler {
	mux := http.NewServeMux()

	// Return the authenticated user's notifications.
	mux.HandleFunc("GET /notifications", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "unauthorized",
			})
			return
		}

		notifications, err := service.List(
			r.Context(),
			claims.CollegeID,
			claims.UserID,
		)

		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid notification request",
			})
			return

		case err != nil:
			log.Printf("list notifications: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "internal server error",
			})
			return
		}

		writeJSON(w, http.StatusOK, notifications)
	})

	// Mark one of the authenticated user's notifications as read.
	mux.HandleFunc("PATCH /notifications/{notification_id}/read", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "unauthorized",
			})
			return
		}

		notificationID, err := strconv.ParseInt(
			r.PathValue("notification_id"),
			10,
			64,
		)
		if err != nil || notificationID <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid notification ID",
			})
			return
		}

		notification, err := service.MarkRead(
			r.Context(),
			claims.CollegeID,
			claims.UserID,
			notificationID,
		)

		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid notification request",
			})
			return

		case errors.Is(err, ErrNotificationNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "notification not found",
			})
			return

		case err != nil:
			log.Printf("mark notification read: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "internal server error",
			})
			return
		}

		writeJSON(w, http.StatusOK, notification)
	})

	// Return the authenticated user's unread notification count.
	mux.HandleFunc("GET /notifications/unread-count", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "unauthorized",
			})
			return
		}

		count, err := service.UnreadCount(
			r.Context(),
			claims.CollegeID,
			claims.UserID,
		)

		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid notification request",
			})
			return

		case err != nil:
			log.Printf("count unread notifications: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "internal server error",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]int64{
			"unread_count": count,
		})
	})

	return mux
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	data any,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("write notification response: %v", err)
	}
}
