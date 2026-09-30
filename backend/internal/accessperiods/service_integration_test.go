package accessperiods

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

func TestServiceIntegration(t *testing.T) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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

	collegeA := createTestCollege(t, ctx, tx, "A")
	collegeB := createTestCollege(t, ctx, tx, "B")

	adminA := createTestUser(t, ctx, tx, collegeA, "college_admin", "admin-a")
	facultyA := createTestUser(t, ctx, tx, collegeA, "faculty", "faculty-a")
	studentA := createTestUser(t, ctx, tx, collegeA, "student", "student-a")
	adminB := createTestUser(t, ctx, tx, collegeB, "college_admin", "admin-b")

	groupID := createTestGroup(t, ctx, tx, collegeA, adminA)
	subgroupID := createTestSubgroup(t, ctx, tx, collegeA, groupID, adminA)

	addMembership(t, ctx, tx, collegeA, groupID, facultyA, "faculty")
	addMembership(t, ctx, tx, collegeA, groupID, studentA, "student")

	_, err = tx.Exec(ctx, `
		INSERT INTO subgroup_faculty_assignments (
			college_id, subgroup_id, faculty_id, assigned_by
		)
		VALUES ($1, $2, $3, $4)
	`, collegeA, subgroupID, facultyA, adminA)
	if err != nil {
		t.Fatalf("assign faculty: %v", err)
	}

	service := NewService(tx)

	start := time.Date(2030, 1, 10, 10, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)

	created, err := service.Create(
		ctx, collegeA, facultyA, "faculty", groupID, subgroupID,
		CreateInput{StartsAt: start, EndsAt: end},
	)
	if err != nil {
		t.Fatalf("create access period: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("expected persisted access period ID")
	}

	_, err = service.Create(
		ctx, collegeA, facultyA, "faculty", groupID, subgroupID,
		CreateInput{
			StartsAt: start.Add(30 * time.Minute),
			EndsAt:   end.Add(time.Hour),
		},
	)
	if err != ErrPeriodOverlap {
		t.Fatalf("expected ErrPeriodOverlap, got %v", err)
	}

	adjacent, err := service.Create(
		ctx, collegeA, adminA, "college_admin", groupID, subgroupID,
		CreateInput{
			StartsAt: end,
			EndsAt:   end.Add(time.Hour),
		},
	)
	if err != nil {
		t.Fatalf("expected adjacent period to be allowed: %v", err)
	}
	if adjacent.ID == 0 {
		t.Fatal("expected adjacent period ID")
	}

	period, err := service.Current(
		ctx, collegeA, studentA, "student", groupID, subgroupID,
		start.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("get current period: %v", err)
	}
	if period == nil || period.ID != created.ID {
		t.Fatalf("expected current period %d, got %+v", created.ID, period)
	}

	period, err = service.Current(
		ctx, collegeA, studentA, "student", groupID, subgroupID,
		end,
	)
	if err != nil {
		t.Fatalf("get boundary current period: %v", err)
	}
	if period == nil || period.ID != adjacent.ID {
		t.Fatalf("expected adjacent period at boundary, got %+v", period)
	}

	open, err := service.IsOpen(
		ctx, collegeA, subgroupID, end.Add(30*time.Minute),
	)
	if err != nil {
		t.Fatalf("check open period: %v", err)
	}
	if !open {
		t.Fatal("expected subgroup to be open during adjacent period")
	}

	open, err = service.IsOpen(
		ctx, collegeA, subgroupID, end.Add(2*time.Hour),
	)
	if err != nil {
		t.Fatalf("check closed period: %v", err)
	}
	if open {
		t.Fatal("expected subgroup to be closed")
	}

	_, err = service.Create(
		ctx, collegeB, adminB, "college_admin", groupID, subgroupID,
		CreateInput{StartsAt: start, EndsAt: end},
	)
	if err != ErrGroupNotFound {
		t.Fatalf("expected cross-college group access to fail, got %v", err)
	}

	_, err = service.Create(
		ctx, collegeA, studentA, "student", groupID, subgroupID,
		CreateInput{StartsAt: start, EndsAt: end},
	)
	if err != ErrForbidden {
		t.Fatalf("expected student create to be forbidden, got %v", err)
	}
}

func createTestCollege(t *testing.T, ctx context.Context, tx pgx.Tx, suffix string) int64 {
	t.Helper()

	var id int64
	code := fmt.Sprintf("AP_%s_%d", suffix, time.Now().UnixNano())
	err := tx.QueryRow(ctx, `
		INSERT INTO colleges (name, code)
		VALUES ($1, $2)
		RETURNING id
	`, "Access Period Test College "+suffix, code).Scan(&id)
	if err != nil {
		t.Fatalf("create college: %v", err)
	}
	return id
}

func createTestUser(t *testing.T, ctx context.Context, tx pgx.Tx, collegeID int64, role, suffix string) int64 {
	t.Helper()

	var id int64
	email := fmt.Sprintf("%s_%d@example.test", suffix, time.Now().UnixNano())
	err := tx.QueryRow(ctx, `
		INSERT INTO users (
			college_id, full_name, email, password_hash, role
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, collegeID, "Access Period "+role, email, "integration-test-hash", role).Scan(&id)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return id
}

func createTestGroup(t *testing.T, ctx context.Context, tx pgx.Tx, collegeID, createdBy int64) int64 {
	t.Helper()

	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO groups (college_id, name, description, created_by)
		VALUES ($1, $2, '', $3)
		RETURNING id
	`, collegeID, fmt.Sprintf("Access Period Group %d", time.Now().UnixNano()), createdBy).Scan(&id)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	return id
}

func createTestSubgroup(t *testing.T, ctx context.Context, tx pgx.Tx, collegeID, groupID, createdBy int64) int64 {
	t.Helper()

	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO subgroups (
			college_id, group_id, name, description, created_by
		)
		VALUES ($1, $2, $3, '', $4)
		RETURNING id
	`, collegeID, groupID, fmt.Sprintf("Access Period Subgroup %d", time.Now().UnixNano()), createdBy).Scan(&id)
	if err != nil {
		t.Fatalf("create subgroup: %v", err)
	}
	return id
}

func addMembership(t *testing.T, ctx context.Context, tx pgx.Tx, collegeID, groupID, userID int64, role string) {
	t.Helper()

	_, err := tx.Exec(ctx, `
		INSERT INTO group_memberships (
			college_id, group_id, user_id, membership_role
		)
		VALUES ($1, $2, $3, $4)
	`, collegeID, groupID, userID, role)
	if err != nil {
		t.Fatalf("add membership: %v", err)
	}
}
