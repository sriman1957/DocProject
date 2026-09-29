package groups

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

func TestServiceListIntegration(t *testing.T) {
	// Integration tests must be explicitly enabled.
	if os.Getenv("DOCPROJECT_INTEGRATION_TESTS") != "1" {
		t.Skip("set DOCPROJECT_INTEGRATION_TESTS=1 to run PostgreSQL integration tests")
	}

	// Refuse to run against any database other than the dedicated test DB.
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

	// Create two isolated colleges.
	var collegeA, collegeB int64

	err = tx.QueryRow(ctx, `
		INSERT INTO colleges (name, code)
		VALUES ('Integration College A', $1)
		RETURNING id
	`, uniqueCode("INTA")).Scan(&collegeA)
	if err != nil {
		t.Fatalf("create college A: %v", err)
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO colleges (name, code)
		VALUES ('Integration College B', $1)
		RETURNING id
	`, uniqueCode("INTB")).Scan(&collegeB)
	if err != nil {
		t.Fatalf("create college B: %v", err)
	}

	// Create one admin in each college.
	adminA := createIntegrationAdmin(t, ctx, tx, collegeA, "a")
	_ = createIntegrationAdmin(t, ctx, tx, collegeB, "b")

	service := NewService(tx)

	// Create a group in college A through the real service.
	created, err := service.Create(ctx, collegeA, adminA, CreateInput{
		Name:        "Integration Active Group",
		Description: "Created by the integration test",
	})
	if err != nil {
		t.Fatalf("create active group: %v", err)
	}

	// Create an inactive group directly.
	_, err = tx.Exec(ctx, `
		INSERT INTO groups (
			college_id,
			name,
			description,
			created_by,
			is_active
		)
		VALUES ($1, 'Integration Inactive Group', '', $2, FALSE)
	`, collegeA, adminA)
	if err != nil {
		t.Fatalf("create inactive group: %v", err)
	}

	// Listing college A should return only its active group.
	groupsA, err := service.List(ctx, collegeA)
	if err != nil {
		t.Fatalf("list college A groups: %v", err)
	}
	if len(groupsA) != 1 {
		t.Fatalf("college A: expected 1 active group, got %d", len(groupsA))
	}
	if groupsA[0].ID != created.ID {
		t.Errorf("expected group ID %d, got %d", created.ID, groupsA[0].ID)
	}

	// Listing college B must not reveal college A's group.
	groupsB, err := service.List(ctx, collegeB)
	if err != nil {
		t.Fatalf("list college B groups: %v", err)
	}
	if len(groupsB) != 0 {
		t.Errorf("college B: expected 0 groups, got %d", len(groupsB))
	}

	// Invalid tenant IDs should be rejected.
	_, err = service.List(ctx, 0)
	if err != ErrInvalidInput {
		t.Errorf("expected ErrInvalidInput for college ID 0, got %v", err)
	}
}

func createIntegrationAdmin(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	collegeID int64,
	suffix string,
) int64 {
	t.Helper()

	var userID int64
	email := fmt.Sprintf(
		"%s-%s@integration.docproject.local",
		suffix,
		uniqueCode("user"),
	)

	err := tx.QueryRow(ctx, `
		INSERT INTO users (
			college_id,
			full_name,
			email,
			password_hash,
			role
		)
		VALUES ($1, $2, $3, $4, 'college_admin')
		RETURNING id
	`,
		collegeID,
		"Integration Admin "+suffix,
		email,
		"integration-test-hash",
	).Scan(&userID)

	if err != nil {
		t.Fatalf("create integration admin: %v", err)
	}

	return userID
}

func uniqueCode(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
