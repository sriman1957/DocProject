package notifications

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidInput         = errors.New("invalid notification input")
	ErrNotificationNotFound = errors.New("notification not found")
)

type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type Service struct {
	db Querier
}

func NewService(database Querier) *Service {
	return &Service{db: database}
}

type Notification struct {
	ID          int64     `json:"id"`
	RecipientID int64     `json:"recipient_user_id"`
	ActorID     int64     `json:"actor_user_id"`
	GroupID     int64     `json:"group_id"`
	SubgroupID  *int64    `json:"subgroup_id"`
	DocumentID  *int64    `json:"document_id"`
	Type        string    `json:"type"`
	Message     string    `json:"message"`
	IsRead      bool      `json:"is_read"`
	CreatedAt   time.Time `json:"created_at"`
}

// List returns notifications belonging to the authenticated user
// within the user's college, newest first.
func (s *Service) List(
	ctx context.Context,
	collegeID int64,
	userID int64,
) ([]Notification, error) {
	if collegeID <= 0 || userID <= 0 {
		return nil, ErrInvalidInput
	}

	const query = `
		SELECT
			n.id,
			n.recipient_user_id,
			n.actor_user_id,
			n.group_id,
			n.subgroup_id,
			n.document_id,
			n.type,
			n.message,
			n.is_read,
			n.created_at
		FROM notifications n
		JOIN users recipient
		  ON recipient.id = n.recipient_user_id
		JOIN groups g
		  ON g.id = n.group_id
		WHERE n.recipient_user_id = $1
		  AND recipient.college_id = $2
		  AND g.college_id = $2
		ORDER BY n.created_at DESC, n.id DESC
	`

	rows, err := s.db.Query(ctx, query, userID, collegeID)
	if err != nil {
		return nil, fmt.Errorf("list notifications: query: %w", err)
	}
	defer rows.Close()

	result := make([]Notification, 0)

	for rows.Next() {
		var notification Notification

		err := rows.Scan(
			&notification.ID,
			&notification.RecipientID,
			&notification.ActorID,
			&notification.GroupID,
			&notification.SubgroupID,
			&notification.DocumentID,
			&notification.Type,
			&notification.Message,
			&notification.IsRead,
			&notification.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("list notifications: scan: %w", err)
		}

		result = append(result, notification)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list notifications: rows: %w", err)
	}

	return result, nil
}

// MarkRead marks one of the user's own notifications as read.
// A notification belonging to another user or college is not exposed.
func (s *Service) MarkRead(
	ctx context.Context,
	collegeID int64,
	userID int64,
	notificationID int64,
) (Notification, error) {
	if collegeID <= 0 || userID <= 0 || notificationID <= 0 {
		return Notification{}, ErrInvalidInput
	}

	const query = `
		UPDATE notifications n
		SET is_read = TRUE
		FROM users recipient, groups g
		WHERE n.id = $1
		  AND n.recipient_user_id = $2
		  AND recipient.id = n.recipient_user_id
		  AND recipient.college_id = $3
		  AND g.id = n.group_id
		  AND g.college_id = $3
		RETURNING
			n.id,
			n.recipient_user_id,
			n.actor_user_id,
			n.group_id,
			n.subgroup_id,
			n.document_id,
			n.type,
			n.message,
			n.is_read,
			n.created_at
	`

	var notification Notification

	err := s.db.QueryRow(
		ctx,
		query,
		notificationID,
		userID,
		collegeID,
	).Scan(
		&notification.ID,
		&notification.RecipientID,
		&notification.ActorID,
		&notification.GroupID,
		&notification.SubgroupID,
		&notification.DocumentID,
		&notification.Type,
		&notification.Message,
		&notification.IsRead,
		&notification.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return Notification{}, ErrNotificationNotFound
	}
	if err != nil {
		return Notification{}, fmt.Errorf("mark notification read: update: %w", err)
	}

	return notification, nil
}

// UnreadCount returns the number of unread notifications belonging
// to the specified user within the specified college.
func (s *Service) UnreadCount(
	ctx context.Context,
	collegeID int64,
	userID int64,
) (int64, error) {
	if collegeID <= 0 || userID <= 0 {
		return 0, ErrInvalidInput
	}

	const query = `
		SELECT COUNT(*)
		FROM notifications n
		JOIN users recipient
		  ON recipient.id = n.recipient_user_id
		JOIN groups g
		  ON g.id = n.group_id
		WHERE n.recipient_user_id = $1
		  AND recipient.college_id = $2
		  AND g.college_id = $2
		  AND n.is_read = FALSE
	`

	var count int64

	err := s.db.QueryRow(ctx, query, userID, collegeID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count unread notifications: query: %w", err)
	}

	return count, nil
}
