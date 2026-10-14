package notifications

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type fakeRow struct {
	scan func(dest ...any) error
}

func (r fakeRow) Scan(dest ...any) error {
	return r.scan(dest...)
}

type fakeRows struct {
	values [][]any
	index  int
	err    error
	closed bool
}

func (r *fakeRows) Close() {
	r.closed = true
}

func (r *fakeRows) Err() error {
	return r.err
}

func (r *fakeRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (r *fakeRows) Next() bool {
	return r.index < len(r.values)
}

func (r *fakeRows) Scan(dest ...any) error {
	if r.index >= len(r.values) {
		return errors.New("no rows available")
	}

	values := r.values[r.index]
	if len(values) != len(dest) {
		return errors.New("column count mismatch")
	}

	for i, value := range values {
		switch target := dest[i].(type) {
		case *int64:
			v, ok := value.(int64)
			if !ok {
				return errors.New("invalid int64 value")
			}
			*target = v

		case *string:
			v, ok := value.(string)
			if !ok {
				return errors.New("invalid string value")
			}
			*target = v

		case *bool:
			v, ok := value.(bool)
			if !ok {
				return errors.New("invalid bool value")
			}
			*target = v

		case *time.Time:
			v, ok := value.(time.Time)
			if !ok {
				return errors.New("invalid time value")
			}
			*target = v

		case **int64:
			if value == nil {
				*target = nil
				continue
			}

			v, ok := value.(int64)
			if !ok {
				return errors.New("invalid nullable int64 value")
			}

			copyValue := v
			*target = &copyValue

		default:
			return errors.New("unsupported scan destination")
		}
	}

	r.index++
	return nil
}

func (r *fakeRows) Values() ([]any, error) {
	return nil, errors.New("not implemented")
}

func (r *fakeRows) RawValues() [][]byte {
	return nil
}

func (r *fakeRows) Conn() *pgx.Conn {
	return nil
}

func (r *fakeRows) TypeMap() *pgtype.Map {
	return nil
}

type fakeQuerier struct {
	query    string
	args     []any
	queryErr error
	rowErr   error
	rows     pgx.Rows
}

func (f *fakeQuerier) QueryRow(
	_ context.Context,
	query string,
	args ...any,
) pgx.Row {
	f.query = query
	f.args = args

	return fakeRow{
		scan: func(dest ...any) error {
			if f.rowErr != nil {
				return f.rowErr
			}

			switch len(dest) {
			case 1:
				target, ok := dest[0].(*int64)
				if !ok {
					return errors.New("unexpected count scan destination")
				}
				*target = 3

			case 10:
				*dest[0].(*int64) = 12
				*dest[1].(*int64) = 9
				*dest[2].(*int64) = 4
				*dest[3].(*int64) = 7
				*dest[4].(**int64) = nil
				*dest[5].(**int64) = nil
				*dest[6].(*string) = "subgroup_created"
				*dest[7].(*string) = "A subgroup was created"
				*dest[8].(*bool) = true
				*dest[9].(*time.Time) = time.Date(
					2026, time.October, 9, 10, 0, 0, 0, time.UTC,
				)

			default:
				return errors.New("unexpected scan destination count")
			}

			return nil
		},
	}
}

func (f *fakeQuerier) Query(
	_ context.Context,
	query string,
	args ...any,
) (pgx.Rows, error) {
	f.query = query
	f.args = args

	if f.queryErr != nil {
		return nil, f.queryErr
	}

	return f.rows, nil
}

func TestListSuccess(t *testing.T) {
	now := time.Date(2026, time.October, 9, 10, 0, 0, 0, time.UTC)

	row := []any{
		int64(12),
		int64(9),
		int64(4),
		int64(7),
		nil,
		nil,
		"subgroup_created",
		"A subgroup was created",
		false,
		now,
	}

	rows := &fakeRows{
		values: make([][]any, 1),
	}
	rows.values[0] = row

	db := &fakeQuerier{
		rows: rows,
	}
	service := NewService(db)

	result, err := service.List(context.Background(), 7, 9)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(result))
	}

	if result[0].ID != 12 {
		t.Errorf("expected notification ID 12, got %d", result[0].ID)
	}
	if result[0].RecipientID != 9 {
		t.Errorf("expected recipient ID 9, got %d", result[0].RecipientID)
	}
	if result[0].IsRead {
		t.Error("expected notification to be unread")
	}
	if !result[0].CreatedAt.Equal(now) {
		t.Errorf("expected CreatedAt %v, got %v", now, result[0].CreatedAt)
	}
	if !rows.closed {
		t.Error("expected rows to be closed")
	}
}

func TestListRejectsInvalidIDs(t *testing.T) {
	tests := []struct {
		name      string
		collegeID int64
		userID    int64
	}{
		{"invalid college", 0, 9},
		{"invalid user", 7, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &fakeQuerier{}
			service := NewService(db)

			_, err := service.List(
				context.Background(),
				tt.collegeID,
				tt.userID,
			)

			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
			if db.query != "" {
				t.Fatal("database should not be queried for invalid IDs")
			}
		})
	}
}

func TestListDatabaseError(t *testing.T) {
	db := &fakeQuerier{
		queryErr: errors.New("database unavailable"),
	}
	service := NewService(db)

	_, err := service.List(context.Background(), 7, 9)
	if err == nil {
		t.Fatal("expected database error")
	}
}

func TestMarkReadSuccess(t *testing.T) {
	db := &fakeQuerier{}
	service := NewService(db)

	notification, err := service.MarkRead(
		context.Background(),
		7,
		9,
		12,
	)
	if err != nil {
		t.Fatalf("MarkRead returned error: %v", err)
	}

	if notification.ID != 12 {
		t.Errorf("expected notification ID 12, got %d", notification.ID)
	}
	if notification.RecipientID != 9 {
		t.Errorf("expected recipient ID 9, got %d", notification.RecipientID)
	}
	if !notification.IsRead {
		t.Error("expected notification to be marked read")
	}
	if len(db.args) != 3 {
		t.Fatalf("expected 3 query arguments, got %d", len(db.args))
	}
}

func TestMarkReadNotFound(t *testing.T) {
	db := &fakeQuerier{
		rowErr: pgx.ErrNoRows,
	}
	service := NewService(db)

	_, err := service.MarkRead(context.Background(), 7, 9, 12)
	if !errors.Is(err, ErrNotificationNotFound) {
		t.Fatalf("expected ErrNotificationNotFound, got %v", err)
	}
}

func TestMarkReadRejectsInvalidIDs(t *testing.T) {
	db := &fakeQuerier{}
	service := NewService(db)

	_, err := service.MarkRead(context.Background(), 7, 9, 0)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if db.query != "" {
		t.Fatal("database should not be queried for invalid IDs")
	}
}

func TestUnreadCountSuccess(t *testing.T) {
	db := &fakeQuerier{}
	service := NewService(db)

	count, err := service.UnreadCount(context.Background(), 7, 9)
	if err != nil {
		t.Fatalf("UnreadCount returned error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected unread count 3, got %d", count)
	}
	if len(db.args) != 2 {
		t.Fatalf("expected 2 query arguments, got %d", len(db.args))
	}
}

func TestUnreadCountRejectsInvalidIDs(t *testing.T) {
	db := &fakeQuerier{}
	service := NewService(db)

	_, err := service.UnreadCount(context.Background(), 7, 0)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if db.query != "" {
		t.Fatal("database should not be queried for invalid IDs")
	}
}

func TestUnreadCountDatabaseError(t *testing.T) {
	db := &fakeQuerier{
		rowErr: errors.New("database unavailable"),
	}
	service := NewService(db)

	_, err := service.UnreadCount(context.Background(), 7, 9)
	if err == nil {
		t.Fatal("expected database error")
	}
}
