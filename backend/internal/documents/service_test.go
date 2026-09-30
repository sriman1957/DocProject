package documents

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type mockDB struct {
	queryFunc func(
		ctx context.Context,
		sql string,
		args ...any,
	) (pgx.Rows, error)
}

func (m *mockDB) Query(
	ctx context.Context,
	sql string,
	args ...any,
) (pgx.Rows, error) {
	return m.queryFunc(ctx, sql, args...)
}

type mockRows struct {
	rows         [][]any
	currentIndex int
	err          error
	scanErr      error
}

func (m *mockRows) Close() {}

func (m *mockRows) Err() error {
	return m.err
}

func (m *mockRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (m *mockRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (m *mockRows) Next() bool {
	if m.currentIndex >= len(m.rows) {
		return false
	}

	m.currentIndex++
	return true
}

func (m *mockRows) Scan(dest ...any) error {
	if m.scanErr != nil {
		return m.scanErr
	}

	row := m.rows[m.currentIndex-1]

	if len(dest) != len(row) {
		return errors.New("destination count does not match row count")
	}

	for i := range dest {
		switch target := dest[i].(type) {
		case *int64:
			value, ok := row[i].(int64)
			if !ok {
				return errors.New("invalid int64 value")
			}
			*target = value

		case *string:
			value, ok := row[i].(string)
			if !ok {
				return errors.New("invalid string value")
			}
			*target = value

		case *time.Time:
			value, ok := row[i].(time.Time)
			if !ok {
				return errors.New("invalid time.Time value")
			}
			*target = value

		default:
			return errors.New("unsupported scan destination")
		}
	}

	return nil
}

func (m *mockRows) Values() ([]any, error) {
	if m.currentIndex == 0 || m.currentIndex > len(m.rows) {
		return nil, errors.New("no current row")
	}

	return m.rows[m.currentIndex-1], nil
}

func (m *mockRows) RawValues() [][]byte {
	return nil
}

func (m *mockRows) Conn() *pgx.Conn {
	return nil
}

func (m *mockRows) TypeMap() *pgtype.Map {
	return nil
}

func TestListPersonalVaultSuccess(t *testing.T) {
	t.Parallel()

	expectedUploadedAt := time.Date(
		2026,
		9,
		29,
		10,
		0,
		0,
		0,
		time.UTC,
	)

	mock := &mockDB{
		queryFunc: func(
			ctx context.Context,
			sql string,
			args ...any,
		) (pgx.Rows, error) {
			if len(args) != 2 {
				t.Fatalf("expected 2 query arguments, got %d", len(args))
			}

			if args[0] != int64(1) {
				t.Errorf("expected college ID 1, got %v", args[0])
			}

			if args[1] != int64(10) {
				t.Errorf("expected student ID 10, got %v", args[1])
			}

			return &mockRows{
				rows: [][]any{
					{
						int64(100),
						"degree.pdf",
						"application/pdf",
						int64(245678),
						"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
						expectedUploadedAt,
					},
				},
			}, nil
		},
	}

	service := NewService(mock)

	documents, err := service.ListPersonalVault(
		context.Background(),
		1,
		10,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(documents) != 1 {
		t.Fatalf("expected 1 document, got %d", len(documents))
	}

	if documents[0].ID != 100 {
		t.Errorf("expected document ID 100, got %d", documents[0].ID)
	}

	if documents[0].OriginalFilename != "degree.pdf" {
		t.Errorf(
			"expected filename degree.pdf, got %s",
			documents[0].OriginalFilename,
		)
	}

	if documents[0].MIMEType != "application/pdf" {
		t.Errorf(
			"expected MIME type application/pdf, got %s",
			documents[0].MIMEType,
		)
	}

	if documents[0].FileSizeBytes != 245678 {
		t.Errorf(
			"expected file size 245678, got %d",
			documents[0].FileSizeBytes,
		)
	}

	if documents[0].SHA256 !=
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Errorf(
			"unexpected SHA256: %s",
			documents[0].SHA256,
		)
	}

	if !documents[0].UploadedAt.Equal(expectedUploadedAt) {
		t.Errorf(
			"unexpected uploaded_at: %s",
			documents[0].UploadedAt,
		)
	}
}

func TestListPersonalVaultEmpty(t *testing.T) {
	t.Parallel()

	mock := &mockDB{
		queryFunc: func(
			ctx context.Context,
			sql string,
			args ...any,
		) (pgx.Rows, error) {
			return &mockRows{
				rows: [][]any{},
			}, nil
		},
	}

	service := NewService(mock)

	documents, err := service.ListPersonalVault(
		context.Background(),
		1,
		10,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if documents == nil {
		t.Fatal("expected non-nil empty slice")
	}

	if len(documents) != 0 {
		t.Fatalf(
			"expected 0 documents, got %d",
			len(documents),
		)
	}
}

func TestListPersonalVaultInvalidInput(t *testing.T) {
	t.Parallel()

	service := NewService(&mockDB{})

	tests := []struct {
		name      string
		collegeID int64
		studentID int64
	}{
		{
			name:      "invalid college ID",
			collegeID: 0,
			studentID: 10,
		},
		{
			name:      "negative college ID",
			collegeID: -1,
			studentID: 10,
		},
		{
			name:      "invalid student ID",
			collegeID: 1,
			studentID: 0,
		},
		{
			name:      "negative student ID",
			collegeID: 1,
			studentID: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			documents, err := service.ListPersonalVault(
				context.Background(),
				tt.collegeID,
				tt.studentID,
			)

			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf(
					"expected ErrInvalidInput, got %v",
					err,
				)
			}

			if documents != nil {
				t.Fatalf(
					"expected nil documents, got %#v",
					documents,
				)
			}
		})
	}
}

func TestListPersonalVaultQueryError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("database query failed")

	mock := &mockDB{
		queryFunc: func(
			ctx context.Context,
			sql string,
			args ...any,
		) (pgx.Rows, error) {
			return nil, expectedErr
		},
	}

	service := NewService(mock)

	documents, err := service.ListPersonalVault(
		context.Background(),
		1,
		10,
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected wrapped query error, got %v",
			err,
		)
	}

	if documents != nil {
		t.Fatalf(
			"expected nil documents, got %#v",
			documents,
		)
	}
}

func TestListPersonalVaultScanError(t *testing.T) {
	t.Parallel()

	mock := &mockDB{
		queryFunc: func(
			ctx context.Context,
			sql string,
			args ...any,
		) (pgx.Rows, error) {
			return &mockRows{
				rows: [][]any{
					{
						int64(100),
						"degree.pdf",
						"application/pdf",
						int64(245678),
						"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
						time.Now(),
					},
				},
				scanErr: errors.New("scan failed"),
			}, nil
		},
	}

	service := NewService(mock)

	documents, err := service.ListPersonalVault(
		context.Background(),
		1,
		10,
	)

	if err == nil {
		t.Fatal("expected scan error, got nil")
	}

	if documents != nil {
		t.Fatalf(
			"expected nil documents, got %#v",
			documents,
		)
	}
}

func TestListPersonalVaultRowsError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("rows iteration failed")

	mock := &mockDB{
		queryFunc: func(
			ctx context.Context,
			sql string,
			args ...any,
		) (pgx.Rows, error) {
			return &mockRows{
				rows: [][]any{
					{
						int64(100),
						"degree.pdf",
						"application/pdf",
						int64(245678),
						"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
						time.Now(),
					},
				},
				err: expectedErr,
			}, nil
		},
	}

	service := NewService(mock)

	documents, err := service.ListPersonalVault(
		context.Background(),
		1,
		10,
	)

	if err == nil {
		t.Fatal("expected rows error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected wrapped rows error, got %v",
			err,
		)
	}

	if documents != nil {
		t.Fatalf(
			"expected nil documents, got %#v",
			documents,
		)
	}
}
