package documents

import (
	"context"
	"errors"
	"strings"
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

	queryRowFunc func(
		ctx context.Context,
		sql string,
		args ...any,
	) pgx.Row
}

func (m *mockDB) Query(
	ctx context.Context,
	sql string,
	args ...any,
) (pgx.Rows, error) {
	if m.queryFunc == nil {
		return nil, errors.New("query function not configured")
	}

	return m.queryFunc(ctx, sql, args...)
}

func (m *mockDB) QueryRow(
	ctx context.Context,
	sql string,
	args ...any,
) pgx.Row {
	if m.queryRowFunc == nil {
		return &mockRow{
			err: errors.New(
				"query row function not configured",
			),
		}
	}

	return m.queryRowFunc(ctx, sql, args...)
}

type mockRow struct {
	values []any
	err    error
}

func (m *mockRow) Scan(dest ...any) error {
	if m.err != nil {
		return m.err
	}

	if len(dest) != len(m.values) {
		return errors.New(
			"destination count does not match value count",
		)
	}

	for i := range dest {
		switch target := dest[i].(type) {
		case *int64:
			value, ok := m.values[i].(int64)
			if !ok {
				return errors.New(
					"invalid int64 value",
				)
			}

			*target = value

		case *string:
			value, ok := m.values[i].(string)
			if !ok {
				return errors.New(
					"invalid string value",
				)
			}

			*target = value

		case *time.Time:
			value, ok := m.values[i].(time.Time)
			if !ok {
				return errors.New(
					"invalid time.Time value",
				)
			}

			*target = value

		default:
			return errors.New(
				"unsupported scan destination",
			)
		}
	}

	return nil
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
		return errors.New(
			"destination count does not match row count",
		)
	}

	for i := range dest {
		switch target := dest[i].(type) {
		case *int64:
			value, ok := row[i].(int64)
			if !ok {
				return errors.New(
					"invalid int64 value",
				)
			}

			*target = value

		case *string:
			value, ok := row[i].(string)
			if !ok {
				return errors.New(
					"invalid string value",
				)
			}

			*target = value

		case *time.Time:
			value, ok := row[i].(time.Time)
			if !ok {
				return errors.New(
					"invalid time.Time value",
				)
			}

			*target = value

		default:
			return errors.New(
				"unsupported scan destination",
			)
		}
	}

	return nil
}

func (m *mockRows) Values() ([]any, error) {
	if m.currentIndex == 0 ||
		m.currentIndex > len(m.rows) {
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

type mockStorage struct {
	saveFunc func(
		key string,
		data []byte,
	) error

	deleteFunc func(
		key string,
	) error

	savedKey   string
	savedData  []byte
	deletedKey string
}

func (m *mockStorage) Save(
	key string,
	data []byte,
) error {
	m.savedKey = key
	m.savedData = append(
		[]byte(nil),
		data...,
	)

	if m.saveFunc != nil {
		return m.saveFunc(key, data)
	}

	return nil
}

func (m *mockStorage) Delete(
	key string,
) error {
	m.deletedKey = key

	if m.deleteFunc != nil {
		return m.deleteFunc(key)
	}

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
				t.Fatalf(
					"expected 2 query arguments, got %d",
					len(args),
				)
			}

			if args[0] != int64(1) {
				t.Errorf(
					"expected college ID 1, got %v",
					args[0],
				)
			}

			if args[1] != int64(10) {
				t.Errorf(
					"expected student ID 10, got %v",
					args[1],
				)
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
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if len(documents) != 1 {
		t.Fatalf(
			"expected 1 document, got %d",
			len(documents),
		)
	}

	if documents[0].ID != 100 {
		t.Errorf(
			"expected document ID 100, got %d",
			documents[0].ID,
		)
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

func TestServiceGetPersonalVaultDocumentSuccess(t *testing.T) {
	now := time.Date(
		2026,
		9,
		30,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	db := &mockDB{
		queryRowFunc: func(
			ctx context.Context,
			sql string,
			args ...any,
		) pgx.Row {
			return &mockRow{
				values: []any{
					int64(10),
					"certificate.pdf",
					"application/pdf",
					int64(1024),
					"abc123",
					now,
				},
			}
		},
	}

	service := NewService(db)

	document, err := service.GetPersonalVaultDocument(
		context.Background(),
		1,
		2,
		10,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if document.ID != 10 {
		t.Fatalf("expected document ID 10, got %d", document.ID)
	}

	if document.OriginalFilename != "certificate.pdf" {
		t.Fatalf(
			"expected filename certificate.pdf, got %q",
			document.OriginalFilename,
		)
	}

	if document.MIMEType != "application/pdf" {
		t.Fatalf(
			"expected MIME type application/pdf, got %q",
			document.MIMEType,
		)
	}

	if document.FileSizeBytes != 1024 {
		t.Fatalf(
			"expected file size 1024, got %d",
			document.FileSizeBytes,
		)
	}

	if document.SHA256 != "abc123" {
		t.Fatalf(
			"expected SHA256 abc123, got %q",
			document.SHA256,
		)
	}

	if !document.UploadedAt.Equal(now) {
		t.Fatalf(
			"expected uploaded time %v, got %v",
			now,
			document.UploadedAt,
		)
	}
}

func TestServiceGetPersonalVaultDocumentInvalidInput(t *testing.T) {
	service := NewService(&mockDB{})

	testCases := []struct {
		name       string
		collegeID  int64
		studentID  int64
		documentID int64
	}{
		{
			name:       "invalid college ID",
			collegeID:  0,
			studentID:  2,
			documentID: 10,
		},
		{
			name:       "invalid student ID",
			collegeID:  1,
			studentID:  0,
			documentID: 10,
		},
		{
			name:       "invalid document ID",
			collegeID:  1,
			studentID:  2,
			documentID: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.GetPersonalVaultDocument(
				context.Background(),
				tc.collegeID,
				tc.studentID,
				tc.documentID,
			)

			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf(
					"expected ErrInvalidInput, got %v",
					err,
				)
			}
		})
	}
}

func TestServiceGetPersonalVaultDocument(t *testing.T) {
	t.Parallel()

	uploadedAt := time.Date(
		2026,
		9,
		30,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	rows := &mockRow{
		values: []any{
			int64(42),
			"certificate.pdf",
			"application/pdf",
			int64(1024),
			"abc123",
			uploadedAt,
		},
	}

	db := &mockDB{
		queryRowFunc: func(
			ctx context.Context,
			sql string,
			args ...any,
		) pgx.Row {
			if len(args) != 3 {
				t.Fatalf(
					"expected 3 query arguments, got %d",
					len(args),
				)
			}

			if args[0] != int64(42) {
				t.Fatalf(
					"expected document ID 42, got %v",
					args[0],
				)
			}

			if args[1] != int64(1) {
				t.Fatalf(
					"expected college ID 1, got %v",
					args[1],
				)
			}

			if args[2] != int64(7) {
				t.Fatalf(
					"expected student ID 7, got %v",
					args[2],
				)
			}

			if !strings.Contains(
				sql,
				"AND college_id = $2",
			) {
				t.Fatal(
					"expected query to enforce college isolation",
				)
			}

			if !strings.Contains(
				sql,
				"AND owner_id = $3",
			) {
				t.Fatal(
					"expected query to enforce document ownership",
				)
			}

			if !strings.Contains(
				sql,
				"AND subgroup_id IS NULL",
			) {
				t.Fatal(
					"expected query to restrict documents to personal vault",
				)
			}

			if !strings.Contains(
				sql,
				"AND deleted_at IS NULL",
			) {
				t.Fatal(
					"expected query to exclude deleted documents",
				)
			}

			return rows
		},
	}

	service := NewService(db)

	document, err := service.GetPersonalVaultDocument(
		context.Background(),
		1,
		7,
		42,
	)
	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if document.ID != 42 {
		t.Fatalf(
			"expected document ID 42, got %d",
			document.ID,
		)
	}

	if document.OriginalFilename != "certificate.pdf" {
		t.Fatalf(
			"expected filename certificate.pdf, got %q",
			document.OriginalFilename,
		)
	}

	if document.MIMEType != "application/pdf" {
		t.Fatalf(
			"expected MIME type application/pdf, got %q",
			document.MIMEType,
		)
	}

	if document.FileSizeBytes != 1024 {
		t.Fatalf(
			"expected file size 1024, got %d",
			document.FileSizeBytes,
		)
	}

	if document.SHA256 != "abc123" {
		t.Fatalf(
			"expected SHA256 abc123, got %q",
			document.SHA256,
		)
	}

	if !document.UploadedAt.Equal(uploadedAt) {
		t.Fatalf(
			"expected uploaded_at %v, got %v",
			uploadedAt,
			document.UploadedAt,
		)
	}
}

func TestServiceGetPersonalVaultDocumentNotFound(t *testing.T) {
	db := &mockDB{
		queryRowFunc: func(
			ctx context.Context,
			sql string,
			args ...any,
		) pgx.Row {
			return &mockRow{
				err: pgx.ErrNoRows,
			}
		},
	}

	service := NewService(db)

	_, err := service.GetPersonalVaultDocument(
		context.Background(),
		1,
		2,
		10,
	)

	if !errors.Is(err, ErrDocumentNotFound) {
		t.Fatalf(
			"expected ErrDocumentNotFound, got %v",
			err,
		)
	}
}

func TestServiceGetPersonalVaultDocumentQueryError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New(
		"database query failed",
	)

	db := &mockDB{
		queryRowFunc: func(
			ctx context.Context,
			sql string,
			args ...any,
		) pgx.Row {
			return &mockRow{
				err: expectedErr,
			}
		},
	}

	service := NewService(db)

	_, err := service.GetPersonalVaultDocument(
		context.Background(),
		1,
		2,
		10,
	)

	if err == nil {
		t.Fatal("expected database error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected wrapped database error, got %v",
			err,
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
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if documents == nil {
		t.Fatal(
			"expected non-nil empty slice",
		)
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

			if !errors.Is(
				err,
				ErrInvalidInput,
			) {
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

	expectedErr := errors.New(
		"database query failed",
	)

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

	expectedErr := errors.New(
		"rows iteration failed",
	)

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

func TestCreatePersonalVaultDocumentSuccess(t *testing.T) {
	t.Parallel()

	uploadedAt := time.Date(
		2026,
		9,
		30,
		10,
		0,
		0,
		0,
		time.UTC,
	)

	data := append(
		[]byte("%PDF-1.7\n"),
		[]byte("certificate")...,
	)

	storage := &mockStorage{}

	mock := &mockDB{
		queryRowFunc: func(
			ctx context.Context,
			sql string,
			args ...any,
		) pgx.Row {
			if len(args) != 7 {
				t.Fatalf(
					"expected 7 arguments, got %d",
					len(args),
				)
			}

			if args[0] != int64(1) {
				t.Errorf(
					"expected college ID 1, got %v",
					args[0],
				)
			}

			if args[1] != int64(10) {
				t.Errorf(
					"expected student ID 10, got %v",
					args[1],
				)
			}

			if args[2] != "certificate.pdf" {
				t.Errorf(
					"unexpected filename: %v",
					args[2],
				)
			}

			if args[4] != MIMEPDF {
				t.Errorf(
					"expected PDF MIME type, got %v",
					args[4],
				)
			}

			if args[5] != int64(len(data)) {
				t.Errorf(
					"unexpected file size: %v",
					args[5],
				)
			}

			return &mockRow{
				values: []any{
					int64(100),
					"certificate.pdf",
					MIMEPDF,
					int64(len(data)),
					CalculateSHA256(data),
					uploadedAt,
				},
			}
		},
	}

	service := NewServiceWithStorage(
		mock,
		storage,
	)

	document, err := service.CreatePersonalVaultDocument(
		context.Background(),
		1,
		10,
		"certificate.pdf",
		data,
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if document.ID != 100 {
		t.Errorf(
			"expected ID 100, got %d",
			document.ID,
		)
	}

	if document.MIMEType != MIMEPDF {
		t.Errorf(
			"expected MIME type %s, got %s",
			MIMEPDF,
			document.MIMEType,
		)
	}

	if document.SHA256 != CalculateSHA256(data) {
		t.Errorf(
			"unexpected SHA-256: %s",
			document.SHA256,
		)
	}

	if storage.savedKey == "" {
		t.Fatal(
			"expected storage key to be saved",
		)
	}

	if !IsSafeStorageKey(storage.savedKey) {
		t.Fatalf(
			"generated storage key is unsafe: %s",
			storage.savedKey,
		)
	}

	if string(storage.savedData) != string(data) {
		t.Fatal(
			"stored data does not match uploaded data",
		)
	}
}

func TestCreatePersonalVaultDocumentRejectsInvalidFile(
	t *testing.T,
) {
	t.Parallel()

	storage := &mockStorage{
		saveFunc: func(
			key string,
			data []byte,
		) error {
			t.Fatal(
				"storage should not be called",
			)
			return nil
		},
	}

	mock := &mockDB{
		queryRowFunc: func(
			ctx context.Context,
			sql string,
			args ...any,
		) pgx.Row {
			t.Fatal(
				"database should not be called",
			)
			return nil
		},
	}

	service := NewServiceWithStorage(
		mock,
		storage,
	)

	_, err := service.CreatePersonalVaultDocument(
		context.Background(),
		1,
		10,
		"certificate.pdf",
		[]byte("not a PDF"),
	)

	if !errors.Is(
		err,
		ErrUnsupportedType,
	) {
		t.Fatalf(
			"expected ErrUnsupportedType, got %v",
			err,
		)
	}
}

func TestCreatePersonalVaultDocumentDatabaseFailureCleansUp(
	t *testing.T,
) {
	t.Parallel()

	expectedErr := errors.New(
		"database failure",
	)

	storage := &mockStorage{}

	mock := &mockDB{
		queryRowFunc: func(
			ctx context.Context,
			sql string,
			args ...any,
		) pgx.Row {
			return &mockRow{
				err: expectedErr,
			}
		},
	}

	service := NewServiceWithStorage(
		mock,
		storage,
	)

	data := append(
		[]byte("%PDF-1.7\n"),
		[]byte("certificate")...,
	)

	_, err := service.CreatePersonalVaultDocument(
		context.Background(),
		1,
		10,
		"certificate.pdf",
		data,
	)

	if err == nil {
		t.Fatal("expected database error")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected database error, got %v",
			err,
		)
	}

	if storage.savedKey == "" {
		t.Fatal(
			"expected storage Save to be called",
		)
	}

	if storage.deletedKey != storage.savedKey {
		t.Fatalf(
			"expected cleanup of %q, got %q",
			storage.savedKey,
			storage.deletedKey,
		)
	}
}

func TestCreatePersonalVaultDocumentInvalidInput(
	t *testing.T,
) {
	t.Parallel()

	storage := &mockStorage{}

	service := NewServiceWithStorage(
		&mockDB{},
		storage,
	)

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
			_, err := service.CreatePersonalVaultDocument(
				context.Background(),
				tt.collegeID,
				tt.studentID,
				"certificate.pdf",
				[]byte("%PDF-1.7\ncertificate"),
			)

			if !errors.Is(
				err,
				ErrInvalidInput,
			) {
				t.Fatalf(
					"expected ErrInvalidInput, got %v",
					err,
				)
			}
		})
	}
}
