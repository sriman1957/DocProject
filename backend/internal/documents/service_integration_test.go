package documents

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"testing"
	"time"

	"docproject/backend/db"
	"docproject/backend/internal/accessperiods"
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

func TestCopyPersonalVaultDocumentIntegration(t *testing.T) {
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

	collegeID := createDocumentTestCollege(
		t,
		ctx,
		tx,
		"FEATURE10",
	)

	studentID := createDocumentTestUser(
		t,
		ctx,
		tx,
		collegeID,
		"feature10-student",
		"student",
	)

	adminID := createDocumentTestUser(
		t,
		ctx,
		tx,
		collegeID,
		"feature10-admin",
		"college_admin",
	)

	var groupID int64

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
		collegeID,
		"Feature 10 Group",
		"Feature 10 integration test group",
		adminID,
	).Scan(&groupID)

	if err != nil {
		t.Fatalf("create feature 10 group: %v", err)
	}

	var subgroupID int64

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
		collegeID,
		groupID,
		"Feature 10 Subgroup",
		"Feature 10 integration test subgroup",
		adminID,
	).Scan(&subgroupID)

	if err != nil {
		t.Fatalf("create feature 10 subgroup: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO group_memberships (
			college_id,
			group_id,
			user_id,
			membership_role
		)
		VALUES ($1, $2, $3, 'student')
	`,
		collegeID,
		groupID,
		studentID,
	)

	if err != nil {
		t.Fatalf("create feature 10 student membership: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO subgroup_access_periods (
			subgroup_id,
			college_id,
			starts_at,
			ends_at,
			created_by
		)
		VALUES (
			$1,
			$2,
			NOW() - INTERVAL '1 hour',
			NOW() + INTERVAL '1 hour',
			$3
		)
	`,
		subgroupID,
		collegeID,
		adminID,
	)

	if err != nil {
		t.Fatalf("create feature 10 access period: %v", err)
	}

	storageRoot := t.TempDir()

	fileStorage, err := NewFileStorage(storageRoot)
	if err != nil {
		t.Fatalf("create test file storage: %v", err)
	}

	sourceBytes := []byte(
		"%PDF-1.4\nFeature 10 integration test document\n%%EOF\n",
	)

	sourceHash := fmt.Sprintf("%x", sha256.Sum256(sourceBytes))

	sourceKey := fmt.Sprintf(
		"documents/%s",
		documentUniqueCode("feature10-source"),
	)

	if err := fileStorage.Save(sourceKey, sourceBytes); err != nil {
		t.Fatalf("save source document: %v", err)
	}

	var sourceDocumentID int64

	err = tx.QueryRow(ctx, `
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
		VALUES ($1, $2, NULL, $3, $4, $5, $6, $7)
		RETURNING id
	`,
		collegeID,
		studentID,
		"feature10.pdf",
		sourceKey,
		"application/pdf",
		int64(len(sourceBytes)),
		sourceHash,
	).Scan(&sourceDocumentID)

	if err != nil {
		t.Fatalf("create feature 10 source document: %v", err)
	}

	authorizer := accessperiods.NewService(tx)

	service := NewServiceWithStorageAndAuthorizer(
		tx,
		fileStorage,
		authorizer,
	)

	copiedDocument, err := service.CopyPersonalVaultDocument(
		ctx,
		collegeID,
		studentID,
		sourceDocumentID,
		groupID,
		subgroupID,
	)

	if err != nil {
		t.Fatalf("copy personal vault document: %v", err)
	}

	copiedDocumentID := copiedDocument.ID

	if copiedDocumentID == sourceDocumentID {
		t.Fatal("copied document must have a different ID")
	}

	var (
		copiedFilename       string
		copiedStorageKey     string
		copiedMIME           string
		copiedSize           int64
		copiedHash           string
		copiedSubgroupID     *int64
		copiedOwnerID        int64
		copiedFromDocumentID *int64
		copiedDeletedAt      any
	)

	err = tx.QueryRow(ctx, `
		SELECT
			original_filename,
			storage_key,
			mime_type,
			file_size_bytes,
			sha256,
			subgroup_id,
			owner_id,
			copied_from_document_id,
			deleted_at
		FROM documents
		WHERE id = $1
	`,
		copiedDocumentID,
	).Scan(
		&copiedFilename,
		&copiedStorageKey,
		&copiedMIME,
		&copiedSize,
		&copiedHash,
		&copiedSubgroupID,
		&copiedOwnerID,
		&copiedFromDocumentID,
		&copiedDeletedAt,
	)

	if err != nil {
		t.Fatalf("query copied document: %v", err)
	}

	if copiedFilename != "feature10.pdf" {
		t.Errorf("expected copied filename feature10.pdf, got %q", copiedFilename)
	}

	if copiedMIME != "application/pdf" {
		t.Errorf("expected copied MIME application/pdf, got %q", copiedMIME)
	}

	if copiedSize != int64(len(sourceBytes)) {
		t.Errorf(
			"expected copied size %d, got %d",
			len(sourceBytes),
			copiedSize,
		)
	}

	if copiedHash != sourceHash {
		t.Errorf(
			"expected copied SHA-256 %q, got %q",
			sourceHash,
			copiedHash,
		)
	}

	if copiedOwnerID != studentID {
		t.Errorf(
			"expected copied owner %d, got %d",
			studentID,
			copiedOwnerID,
		)
	}

	if copiedSubgroupID == nil {
		t.Fatal("copied document must belong to subgroup")
	}

	if *copiedSubgroupID != subgroupID {
		t.Errorf(
			"expected copied subgroup %d, got %d",
			subgroupID,
			*copiedSubgroupID,
		)
	}

	if copiedFromDocumentID == nil {
		t.Fatal("copied_from_document_id must be populated")
	}

	if *copiedFromDocumentID != int64(sourceDocumentID) {
		t.Errorf(
			"expected copied_from_document_id %d, got %d",
			sourceDocumentID,
			*copiedFromDocumentID,
		)
	}

	if copiedDeletedAt != nil {
		t.Fatal("newly copied document must not be deleted")
	}

	sourceFile, err := fileStorage.Open(sourceKey)
	if err != nil {
		t.Fatalf("open original source file: %v", err)
	}

	defer sourceFile.Close()

	actualSourceBytes, err := io.ReadAll(sourceFile)
	if err != nil {
		t.Fatalf("read original source file: %v", err)
	}

	if !bytes.Equal(actualSourceBytes, sourceBytes) {
		t.Fatal("original source file was modified")
	}

	copiedFile, err := fileStorage.Open(copiedStorageKey)
	if err != nil {
		t.Fatalf("open copied file: %v", err)
	}

	defer copiedFile.Close()

	actualCopiedBytes, err := io.ReadAll(copiedFile)
	if err != nil {
		t.Fatalf("read copied file: %v", err)
	}

	if !bytes.Equal(actualCopiedBytes, sourceBytes) {
		t.Fatal("copied file contents do not match source")
	}

	var auditCount int

	err = tx.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM document_audit_logs
		WHERE document_id = $1
		  AND actor_user_id = $2
		  AND action = 'document_copied'
		  AND filename = $3
	`,
		copiedDocumentID,
		studentID,
		"feature10.pdf",
	).Scan(&auditCount)

	if err != nil {
		t.Fatalf("query copy audit record: %v", err)
	}

	if auditCount != 1 {
		t.Fatalf(
			"expected exactly 1 document_copied audit record, got %d",
			auditCount,
		)
	}

	var (
		sourceStorageKey string
		sourceSubgroupID *int64
		sourceCopiedFrom *int64
	)

	err = tx.QueryRow(ctx, `
		SELECT
			storage_key,
			subgroup_id,
			copied_from_document_id
		FROM documents
		WHERE id = $1
	`,
		sourceDocumentID,
	).Scan(
		&sourceStorageKey,
		&sourceSubgroupID,
		&sourceCopiedFrom,
	)

	if err != nil {
		t.Fatalf("query original document after copy: %v", err)
	}

	if sourceStorageKey != sourceKey {
		t.Fatal("original document storage key changed")
	}

	if sourceSubgroupID != nil {
		t.Fatal("original personal vault document was moved into subgroup")
	}

	if sourceCopiedFrom != nil {
		t.Fatal("original document unexpectedly has copied_from_document_id")
	}
}

func TestListSubgroupDocumentsIntegration(t *testing.T) {
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
		"FEATURE19A",
	)

	// College B.
	collegeB := createDocumentTestCollege(
		t,
		ctx,
		tx,
		"FEATURE19B",
	)

	adminA := createDocumentTestUser(
		t,
		ctx,
		tx,
		collegeA,
		"feature19-admin",
		"college_admin",
	)

	facultyMember := createDocumentTestUser(
		t,
		ctx,
		tx,
		collegeA,
		"feature19-faculty-member",
		"faculty",
	)

	facultyNonMember := createDocumentTestUser(
		t,
		ctx,
		tx,
		collegeA,
		"feature19-faculty-nonmember",
		"faculty",
	)

	studentA := createDocumentTestUser(
		t,
		ctx,
		tx,
		collegeA,
		"feature19-student",
		"student",
	)

	studentB := createDocumentTestUser(
		t,
		ctx,
		tx,
		collegeB,
		"feature19-student-b",
		"student",
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
		"Feature 19 Group",
		"Feature 19 subgroup document listing group",
		adminA,
	).Scan(&groupA)

	if err != nil {
		t.Fatalf("create feature 19 group: %v", err)
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
		"Feature 19 Subgroup",
		"Feature 19 subgroup document listing",
		adminA,
	).Scan(&subgroupA)

	if err != nil {
		t.Fatalf("create feature 19 subgroup: %v", err)
	}

	// Faculty member belongs to the requested group.
	_, err = tx.Exec(ctx, `
		INSERT INTO group_memberships (
			college_id,
			group_id,
			user_id,
			membership_role
		)
		VALUES ($1, $2, $3, 'faculty')
	`,
		collegeA,
		groupA,
		facultyMember,
	)

	if err != nil {
		t.Fatalf("create feature 19 faculty membership: %v", err)
	}

	// Student A also belongs to the group.
	_, err = tx.Exec(ctx, `
		INSERT INTO group_memberships (
			college_id,
			group_id,
			user_id,
			membership_role
		)
		VALUES ($1, $2, $3, 'student')
	`,
		collegeA,
		groupA,
		studentA,
	)

	if err != nil {
		t.Fatalf("create feature 19 student membership: %v", err)
	}

	// Active subgroup documents.
	firstDocumentID := createDocumentTestDocument(
		t,
		ctx,
		tx,
		collegeA,
		studentA,
		&subgroupA,
		"first.pdf",
		"application/pdf",
		100,
	)

	secondDocumentID := createDocumentTestDocument(
		t,
		ctx,
		tx,
		collegeA,
		studentA,
		&subgroupA,
		"second.pdf",
		"application/pdf",
		200,
	)

	// Create a group and subgroup in another college.
	var groupB int64

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
		collegeB,
		"Feature 19 Other College Group",
		"Feature 19 cross-tenant test group",
		studentB,
	).Scan(&groupB)

	if err != nil {
		t.Fatalf("create feature 19 other-college group: %v", err)
	}

	var subgroupB int64

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
		collegeB,
		groupB,
		"Feature 19 Other College Subgroup",
		"Feature 19 cross-tenant test subgroup",
		studentB,
	).Scan(&subgroupB)

	if err != nil {
		t.Fatalf("create feature 19 other-college subgroup: %v", err)
	}

	// Document in another college must never leak.
	createDocumentTestDocument(
		t,
		ctx,
		tx,
		collegeB,
		studentB,
		&subgroupB,
		"other-college.pdf",
		"application/pdf",
		300,
	)

	// Deleted subgroup document must not be returned.
	deletedDocumentID := createDocumentTestDocument(
		t,
		ctx,
		tx,
		collegeA,
		studentA,
		&subgroupA,
		"deleted.pdf",
		"application/pdf",
		400,
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
		t.Fatalf("delete feature 19 document: %v", err)
	}

	// Use the service directly with the test transaction.
	service := NewService(tx)

	// College admin can list subgroup documents.
	documents, err := service.ListSubgroupDocuments(
		ctx,
		collegeA,
		adminA,
		"college_admin",
		groupA,
		subgroupA,
	)

	if err != nil {
		t.Fatalf("admin list subgroup documents: %v", err)
	}

	if len(documents) != 2 {
		t.Fatalf(
			"expected 2 active subgroup documents for admin, got %d",
			len(documents),
		)
	}

	// Newest document must appear first.
	if documents[0].ID != secondDocumentID {
		t.Errorf(
			"expected newest document ID %d first, got %d",
			secondDocumentID,
			documents[0].ID,
		)
	}

	if documents[1].ID != firstDocumentID {
		t.Errorf(
			"expected older document ID %d second, got %d",
			firstDocumentID,
			documents[1].ID,
		)
	}

	// Faculty who belongs to the group can list documents.
	documents, err = service.ListSubgroupDocuments(
		ctx,
		collegeA,
		facultyMember,
		"faculty",
		groupA,
		subgroupA,
	)

	if err != nil {
		t.Fatalf("member faculty list subgroup documents: %v", err)
	}

	if len(documents) != 2 {
		t.Fatalf(
			"expected 2 subgroup documents for member faculty, got %d",
			len(documents),
		)
	}

	// Faculty who does not belong to the group must be forbidden.
	_, err = service.ListSubgroupDocuments(
		ctx,
		collegeA,
		facultyNonMember,
		"faculty",
		groupA,
		subgroupA,
	)

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf(
			"expected ErrForbidden for non-member faculty, got %v",
			err,
		)
	}

	// Students must be forbidden.
	_, err = service.ListSubgroupDocuments(
		ctx,
		collegeA,
		studentA,
		"student",
		groupA,
		subgroupA,
	)

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf(
			"expected ErrForbidden for student, got %v",
			err,
		)
	}

	// A subgroup from another group must not be accessible.
	otherGroupID := int64(0)

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
		"Feature 19 Other Group",
		"Feature 19 authorization test",
		adminA,
	).Scan(&otherGroupID)

	if err != nil {
		t.Fatalf("create feature 19 other group: %v", err)
	}

	_, err = service.ListSubgroupDocuments(
		ctx,
		collegeA,
		adminA,
		"college_admin",
		otherGroupID,
		subgroupA,
	)

	if !errors.Is(err, ErrDocumentNotFound) {
		t.Fatalf(
			"expected ErrDocumentNotFound for mismatched group/subgroup, got %v",
			err,
		)
	}
}
