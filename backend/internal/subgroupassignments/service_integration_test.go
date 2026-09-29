package subgroupassignments

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

func TestSubgroupAssignmentServiceIntegration(t *testing.T) {
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

	// Two isolated colleges.
	collegeA := createAssignmentTestCollege(t, ctx, tx, "ASA")
	collegeB := createAssignmentTestCollege(t, ctx, tx, "ASB")

	adminA := createAssignmentTestUser(
		t, ctx, tx, collegeA, "admin-a", "college_admin",
	)
	adminB := createAssignmentTestUser(
		t, ctx, tx, collegeB, "admin-b", "college_admin",
	)

	groupFaculty := createAssignmentTestUser(
		t, ctx, tx, collegeA, "group-faculty", "faculty",
	)
	targetFaculty := createAssignmentTestUser(
		t, ctx, tx, collegeA, "target-faculty", "faculty",
	)
	outsideFaculty := createAssignmentTestUser(
		t, ctx, tx, collegeA, "outside-faculty", "faculty",
	)
	student := createAssignmentTestUser(
		t, ctx, tx, collegeA, "student", "student",
	)
	inactiveFaculty := createAssignmentTestUser(
		t, ctx, tx, collegeA, "inactive-faculty", "faculty",
	)

	// Group in college A.
	var groupA int64
	err = tx.QueryRow(ctx, `
		INSERT INTO groups (college_id, name, description, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`,
		collegeA,
		"Assignment Integration Group",
		"Created for assignment integration tests",
		adminA,
	).Scan(&groupA)
	if err != nil {
		t.Fatalf("create group A: %v", err)
	}

	// A separate group in college B.
	var groupB int64
	err = tx.QueryRow(ctx, `
		INSERT INTO groups (college_id, name, description, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`,
		collegeB,
		"Assignment Isolation Group",
		"Created for tenant isolation tests",
		adminB,
	).Scan(&groupB)
	if err != nil {
		t.Fatalf("create group B: %v", err)
	}

	// Group A memberships.
	_, err = tx.Exec(ctx, `
		INSERT INTO group_memberships (
			college_id, group_id, user_id, membership_role
		)
		VALUES
			($1, $2, $3, 'faculty'),
			($1, $2, $4, 'faculty'),
			($1, $2, $5, 'student')
	`,
		collegeA, groupA, groupFaculty, targetFaculty, student,
	)
	if err != nil {
		t.Fatalf("create group A memberships: %v", err)
	}

	// Make one group member inactive to verify eligibility checks.
	_, err = tx.Exec(ctx, `
		UPDATE users
		SET is_active = FALSE
		WHERE id = $1 AND college_id = $2
	`, inactiveFaculty, collegeA)
	if err != nil {
		t.Fatalf("deactivate faculty: %v", err)
	}

	// Add inactive faculty to the group as well.
	_, err = tx.Exec(ctx, `
		INSERT INTO group_memberships (
			college_id, group_id, user_id, membership_role
		)
		VALUES ($1, $2, $3, 'faculty')
	`, collegeA, groupA, inactiveFaculty)
	if err != nil {
		t.Fatalf("add inactive faculty membership: %v", err)
	}

	// Create one subgroup in each college.
	var subgroupA int64
	err = tx.QueryRow(ctx, `
		INSERT INTO subgroups (
			college_id, group_id, name, description, created_by
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`,
		collegeA, groupA, "Assignment Subgroup A",
		"Subgroup for assignment tests", adminA,
	).Scan(&subgroupA)
	if err != nil {
		t.Fatalf("create subgroup A: %v", err)
	}

	var subgroupB int64
	err = tx.QueryRow(ctx, `
		INSERT INTO subgroups (
			college_id, group_id, name, description, created_by
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`,
		collegeB, groupB, "Assignment Subgroup B",
		"Subgroup for isolation tests", adminB,
	).Scan(&subgroupB)
	if err != nil {
		t.Fatalf("create subgroup B: %v", err)
	}

	service := NewService(tx)

	// 1. A college admin can assign eligible faculty.
	first, err := service.Create(
		ctx, collegeA, adminA, "college_admin",
		groupA, subgroupA, targetFaculty,
	)
	if err != nil {
		t.Fatalf("admin create assignment: %v", err)
	}
	if first.ID <= 0 {
		t.Fatalf("expected valid assignment ID, got %d", first.ID)
	}
	if first.FacultyID != targetFaculty {
		t.Errorf("expected faculty %d, got %d", targetFaculty, first.FacultyID)
	}
	if first.RevokedBy != nil || first.RevokedAt != "" {
		t.Errorf("new assignment should not be revoked: %+v", first)
	}

	// 2. Parent-group faculty can assign another eligible faculty member.
	second, err := service.Create(
		ctx, collegeA, groupFaculty, "faculty",
		groupA, subgroupA, groupFaculty,
	)
	if err != nil {
		t.Fatalf("group faculty create assignment: %v", err)
	}
	if second.FacultyID != groupFaculty {
		t.Errorf("expected faculty %d, got %d", groupFaculty, second.FacultyID)
	}

	// 3. Duplicate active assignment is rejected.
	_, err = service.Create(
		ctx, collegeA, adminA, "college_admin",
		groupA, subgroupA, targetFaculty,
	)
	if err != ErrAssignmentExists {
		t.Errorf("expected ErrAssignmentExists, got %v", err)
	}

	// 4. Faculty outside the group cannot manage assignments.
	_, err = service.Create(
		ctx, collegeA, outsideFaculty, "faculty",
		groupA, subgroupA, targetFaculty,
	)
	if err != ErrForbidden {
		t.Errorf("expected ErrForbidden for outside faculty, got %v", err)
	}

	// 5. Students cannot manage assignments.
	_, err = service.Create(
		ctx, collegeA, student, "student",
		groupA, subgroupA, targetFaculty,
	)
	if err != ErrForbidden {
		t.Errorf("expected ErrForbidden for student, got %v", err)
	}

	// 6. Faculty who are not group members are not eligible targets.
	_, err = service.Create(
		ctx, collegeA, adminA, "college_admin",
		groupA, subgroupA, outsideFaculty,
	)
	if err != ErrFacultyNotEligible {
		t.Errorf("expected ErrFacultyNotEligible, got %v", err)
	}

	// 7. Inactive faculty cannot be assigned.
	_, err = service.Create(
		ctx, collegeA, adminA, "college_admin",
		groupA, subgroupA, inactiveFaculty,
	)
	if err != ErrFacultyNotEligible {
		t.Errorf("expected ErrFacultyNotEligible for inactive user, got %v", err)
	}

	// 8. A student cannot list assignment records.
	_, err = service.List(
		ctx, collegeA, student, "student", groupA, subgroupA,
	)
	if err != ErrForbidden {
		t.Errorf("expected ErrForbidden for student listing, got %v", err)
	}

	// 9. An authorized group faculty member can list assignment history.
	assignments, err := service.List(
		ctx, collegeA, groupFaculty, "faculty", groupA, subgroupA,
	)
	if err != nil {
		t.Fatalf("group faculty list assignments: %v", err)
	}
	if len(assignments) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(assignments))
	}

	// 10. Revoke an active assignment. The row must remain in history.
	revoked, err := service.Revoke(
		ctx, collegeA, adminA, "college_admin",
		groupA, subgroupA, targetFaculty,
	)
	if err != nil {
		t.Fatalf("revoke assignment: %v", err)
	}
	if revoked.RevokedBy == nil || *revoked.RevokedBy != adminA {
		t.Errorf("expected revoked_by %d, got %v", adminA, revoked.RevokedBy)
	}
	if revoked.RevokedAt == "" {
		t.Error("expected revoked_at to be set")
	}

	// 11. Revoking the same assignment again should report not found.
	_, err = service.Revoke(
		ctx, collegeA, adminA, "college_admin",
		groupA, subgroupA, targetFaculty,
	)
	if err != ErrAssignmentNotFound {
		t.Errorf("expected ErrAssignmentNotFound, got %v", err)
	}

	// 12. Reassignment after revocation creates a new record.
	reassigned, err := service.Create(
		ctx, collegeA, adminA, "college_admin",
		groupA, subgroupA, targetFaculty,
	)
	if err != nil {
		t.Fatalf("reassign faculty: %v", err)
	}
	if reassigned.ID == first.ID {
		t.Error("expected reassignment to create a new history record")
	}

	// 13. Listing includes revoked history and the new active assignment.
	history, err := service.List(
		ctx, collegeA, adminA, "college_admin", groupA, subgroupA,
	)
	if err != nil {
		t.Fatalf("list assignment history: %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("expected 3 assignment records, got %d", len(history))
	}

	// 14. College B cannot manage college A's group.
	_, err = service.Create(
		ctx, collegeB, adminB, "college_admin",
		groupA, subgroupA, targetFaculty,
	)
	if err != ErrGroupNotFound {
		t.Errorf("expected ErrGroupNotFound, got %v", err)
	}

	// 15. A subgroup from another group cannot be used.
	_, err = service.Create(
		ctx, collegeA, adminA, "college_admin",
		groupA, subgroupB, targetFaculty,
	)
	if err != ErrSubgroupNotFound {
		t.Errorf("expected ErrSubgroupNotFound, got %v", err)
	}

	// 16. Invalid IDs are rejected.
	_, err = service.Create(
		ctx, 0, adminA, "college_admin",
		groupA, subgroupA, targetFaculty,
	)
	if err != ErrInvalidInput {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func createAssignmentTestCollege(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	prefix string,
) int64 {
	t.Helper()

	var collegeID int64
	err := tx.QueryRow(ctx, `
		INSERT INTO colleges (name, code)
		VALUES ($1, $2)
		RETURNING id
	`,
		"Assignment Integration College",
		assignmentUniqueCode(prefix),
	).Scan(&collegeID)
	if err != nil {
		t.Fatalf("create assignment test college: %v", err)
	}

	return collegeID
}

func createAssignmentTestUser(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	collegeID int64,
	suffix string,
	role string,
) int64 {
	t.Helper()

	email := fmt.Sprintf(
		"%s-%s@integration.docproject.local",
		suffix,
		assignmentUniqueCode("user"),
	)

	var userID int64
	err := tx.QueryRow(ctx, `
		INSERT INTO users (
			college_id, full_name, email, password_hash, role
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`,
		collegeID,
		"Assignment Integration "+suffix,
		email,
		"integration-test-hash",
		role,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("create assignment test user (%s): %v", role, err)
	}

	return userID
}

func assignmentUniqueCode(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
