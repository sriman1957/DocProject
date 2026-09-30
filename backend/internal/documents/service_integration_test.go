package documents

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"docproject/backend/db"
	"docproject/backend/internal/config"

	"github.com/jackc/pgx/v5"
)

func TestDocumentServiceIntegration(t *testing.T) {
	if os.Getenv("DOCPROJECT_INTEGRATION_TESTS") != "1" {
		t.Skip("set DOCPROJECT_INTEGRATION_TESTS=1 to run PostgreSQL integration tests")
	}

	if os.Getenv("APP_ENV") != "test" {
		t.Fatal("integration tests require APP_ENV=test")
	}

	if os.Getenv("DB_NAME") != "docproject_test" {
		t.Fatal("integration tests may only run against DB_NAME=docproject_test")
	}

	port, err := strconv.ParseUint(os.Getenv("DB_PORT"), 10, 16)
	if err != nil || port == 0 {
		t.Fatalf("invalid DB_PORT: %q", os.Getenv("DB_PORT"))
	}

	cfg := config.Config{
		AppEnv:     "test",
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     uint16(port),
		DBName:     "docproject_test",
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
	}

	if cfg.DBHost == "" || cfg.DBUser == "" {
		t.Fatal("DB_HOST and DB_USER must be set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := db.New(cfg)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}

	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	// College A.
	collegeA := createDocumentTestCollege(
		t,
		ctx,
		tx,
		"DOCA",
	)

	// College B.
	collegeB := createDocumentTestCollege(
		t,
		ctx,
		tx,
		"DOCB",
	)

	// Student A owns the documents we want to test.
	studentA := createDocumentTestUser(
		t,
		ctx,
		tx,
		collegeA,
		"student-a",
		"student",
	)

	// Student B is in the same college but owns different documents.
	studentB := createDocumentTestUser(
		t,
		ctx,
		tx,
		collegeA,
		"student-b",
		"student",
	)

	// Student C is in another college.
	studentC := createDocumentTestUser(
		t,
		ctx,
		tx,
		collegeB,
		"student-c",
		"student",
	)

	adminA := createDocumentTestUser(
		t,
		ctx,
		tx,
		collegeA,
		"admin-a",
		"college_admin",
	)

	var groupA int64

	err = tx.QueryRow(ctx, `
		INSERT INTO groups (
			college_id,
			name,
			description,
			created_by
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`,
		collegeA,
		"Document Test Group",
		"Personal vault integration test group",
		adminA,
	).Scan(&groupA)

	if err != nil {
		t.Fatalf("create test group: %v", err)
	}

	var subgroupA int64

	err = tx.QueryRow(ctx, `
		INSERT INTO subgroups (
			college_id,
			group_id,
			name,
			description,
			created_by
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`,
		collegeA,
		groupA,
		"Internship",
		"Document integration test subgroup",
		adminA,
	).Scan(&subgroupA)

	if err != nil {
		t.Fatalf("create test subgroup: %v", err)
	}

	// Student A personal document.
	personalDocumentA := createDocumentTestDocument(
		t,
		ctx,
		tx,
		collegeA,
		studentA,
		nil,
		"degree.pdf",
		"application/pdf",
		245678,
	)

	// Student B personal document.
	createDocumentTestDocument(
		t,
		ctx,
		tx,
		collegeA,
		studentB,
		nil,
		"student-b.pdf",
		"application/pdf",
		123456,
	)

	// Student C personal document in another tenant.
	createDocumentTestDocument(
		t,
		ctx,
		tx,
		collegeB,
		studentC,
		nil,
		"student-c.pdf",
		"application/pdf",
		987654,
	)

	// Student A document inside a subgroup.
	createDocumentTestDocument(
		t,
		ctx,
		tx,
		collegeA,
		studentA,
		&subgroupA,
		"internship.pdf",
		"application/pdf",
		456789,
	)

	// Student A deleted personal document.
	deletedDocumentID := createDocumentTestDocument(
		t,
		ctx,
		tx,
		collegeA,
		studentA,
		nil,
		"deleted.pdf",
		"application/pdf",
		111111,
	)

	_, err = tx.Exec(ctx, `
		UPDATE documents
		SET
			deleted_at = NOW(),
			deleted_by = $1
		WHERE id = $2
		  AND college_id = $3
	`,
		studentA,
		deletedDocumentID,
		collegeA,
	)

	if err != nil {
		t.Fatalf("delete test document: %v", err)
	}

	service := NewService(tx)

	// Student A should see exactly one active personal-vault document.
	documents, err := service.ListPersonalVault(
		ctx,
		collegeA,
		studentA,
	)

	if err != nil {
		t.Fatalf("list student A personal vault: %v", err)
	}

	if len(documents) != 1 {
		t.Fatalf(
			"expected 1 personal document for student A, got %d",
			len(documents),
		)
	}

	if documents[0].ID != personalDocumentA {
		t.Errorf(
			"expected document ID %d, got %d",
			personalDocumentA,
			documents[0].ID,
		)
	}

	if documents[0].OriginalFilename != "degree.pdf" {
		t.Errorf(
			"expected degree.pdf, got %q",
			documents[0].OriginalFilename,
		)
	}

	if documents[0].MIMEType != "application/pdf" {
		t.Errorf(
			"expected application/pdf, got %q",
			documents[0].MIMEType,
		)
	}

	if documents[0].FileSizeBytes != 245678 {
		t.Errorf(
			"expected file size 245678, got %d",
			documents[0].FileSizeBytes,
		)
	}

	// Student B must only see their own document.
	documents, err = service.ListPersonalVault(
		ctx,
		collegeA,
		studentB,
	)

	if err != nil {
		t.Fatalf("list student B personal vault: %v", err)
	}

	if len(documents) != 1 {
		t.Fatalf(
			"expected 1 personal document for student B, got %d",
			len(documents),
		)
	}

	if documents[0].OriginalFilename != "student-b.pdf" {
		t.Errorf(
			"student B received unexpected document %q",
			documents[0].OriginalFilename,
		)
	}

	// Student C belongs to another college and must only see
	// their own tenant's document.
	documents, err = service.ListPersonalVault(
		ctx,
		collegeB,
		studentC,
	)

	if err != nil {
		t.Fatalf("list student C personal vault: %v", err)
	}

	if len(documents) != 1 {
		t.Fatalf(
			"expected 1 personal document for student C, got %d",
			len(documents),
		)
	}

	if documents[0].OriginalFilename != "student-c.pdf" {
		t.Errorf(
			"student C received unexpected document %q",
			documents[0].OriginalFilename,
		)
	}

	// Cross-tenant ownership must not leak documents.
	documents, err = service.ListPersonalVault(
		ctx,
		collegeA,
		studentC,
	)

	if err != nil {
		t.Fatalf("cross-tenant personal vault query: %v", err)
	}

	if len(documents) != 0 {
		t.Fatalf(
			"expected 0 documents for cross-tenant query, got %d",
			len(documents),
		)
	}

	// Re-query Student A to explicitly verify that subgroup and
	// deleted documents are excluded from the personal vault.
	documents, err = service.ListPersonalVault(
		ctx,
		collegeA,
		studentA,
	)

	if err != nil {
		t.Fatalf("re-list student A personal vault: %v", err)
	}

	for _, document := range documents {
		if document.OriginalFilename == "internship.pdf" {
			t.Fatal("subgroup document leaked into personal vault")
		}

		if document.OriginalFilename == "deleted.pdf" {
			t.Fatal("deleted document leaked into personal vault")
		}
	}
}

func createDocumentTestCollege(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	prefix string,
) int64 {
	t.Helper()

	var collegeID int64

	err := tx.QueryRow(ctx, `
		INSERT INTO colleges (
			name,
			code
		)
		VALUES ($1, $2)
		RETURNING id
	`,
		"Document Integration College",
		documentUniqueCode(prefix),
	).Scan(&collegeID)

	if err != nil {
		t.Fatalf("create document test college: %v", err)
	}

	return collegeID
}

func createDocumentTestUser(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	collegeID int64,
	suffix string,
	role string,
) int64 {
	t.Helper()

	var userID int64

	email := fmt.Sprintf(
		"%s-%s@integration.docproject.local",
		suffix,
		documentUniqueCode("user"),
	)

	err := tx.QueryRow(ctx, `
		INSERT INTO users (
			college_id,
			full_name,
			email,
			password_hash,
			role
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`,
		collegeID,
		"Document Integration "+suffix,
		email,
		"integration-test-hash",
		role,
	).Scan(&userID)

	if err != nil {
		t.Fatalf(
			"create document test user (%s): %v",
			role,
			err,
		)
	}

	return userID
}

func createDocumentTestDocument(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	collegeID int64,
	ownerID int64,
	subgroupID *int64,
	filename string,
	mimeType string,
	fileSize int64,
) int64 {
	t.Helper()

	var documentID int64

	storageKey := fmt.Sprintf(
		"test/%s/%s",
		documentUniqueCode("storage"),
		filename,
	)

	sha256 := documentTestSHA256(filename)

	err := tx.QueryRow(ctx, `
		INSERT INTO documents (
			college_id,
			owner_id,
			subgroup_id,
			original_filename,
			storage_key,
			mime_type,
			file_size_bytes,
			sha256
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`,
		collegeID,
		ownerID,
		subgroupID,
		filename,
		storageKey,
		mimeType,
		fileSize,
		sha256,
	).Scan(&documentID)

	if err != nil {
		t.Fatalf(
			"create document %q: %v",
			filename,
			err,
		)
	}

	return documentID
}

func documentTestSHA256(filename string) string {
	switch filename {
	case "degree.pdf":
		return "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	case "student-b.pdf":
		return "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	case "student-c.pdf":
		return "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	case "internship.pdf":
		return "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	case "deleted.pdf":
		return "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	default:
		return "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	}
}

func documentUniqueCode(prefix string) string {
	return fmt.Sprintf(
		"%s_%d",
		prefix,
		time.Now().UnixNano(),
	)
}
