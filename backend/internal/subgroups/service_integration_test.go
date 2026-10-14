package subgroups

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

func TestSubgroupServiceIntegration(t *testing.T) {
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

	// Create two isolated colleges.
	collegeA := createSubgroupTestCollege(t, ctx, tx, "SGA")
	collegeB := createSubgroupTestCollege(t, ctx, tx, "SGB")

	// Create users in college A.
	adminA := createSubgroupTestUser(
		t, ctx, tx, collegeA, "admin", "college_admin",
	)
	facultyA := createSubgroupTestUser(
		t, ctx, tx, collegeA, "faculty", "faculty",
	)
	studentA := createSubgroupTestUser(
		t, ctx, tx, collegeA, "student", "student",
	)
	inactiveStudentA := createSubgroupTestUser(
		t, ctx, tx, collegeA, "inactive-student", "student",
	)

	_, err = tx.Exec(ctx, `
		UPDATE users
		SET is_active = FALSE
		WHERE id = $1
	`, inactiveStudentA)
	if err != nil {
		t.Fatalf("deactivate integration test student: %v", err)
	}
	outsiderFaculty := createSubgroupTestUser(
		t, ctx, tx, collegeA, "outsider", "faculty",
	)

	// Create an administrator in college B.
	adminB := createSubgroupTestUser(
		t, ctx, tx, collegeB, "admin", "college_admin",
	)

	// Create a student in college B.
	studentB := createSubgroupTestUser(
		t, ctx, tx, collegeB, "student", "student",
	)

	// Create a group in college A.
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
		"Subgroup Integration Group",
		"Created for subgroup integration tests",
		adminA,
	).Scan(&groupA)
	if err != nil {
		t.Fatalf("create group A: %v", err)
	}

	// Create a separate group in college B with its own student.
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
		"Subgroup Integration Group B",
		"Created for notification isolation tests",
		adminB,
	).Scan(&groupB)
	if err != nil {
		t.Fatalf("create group B: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO group_memberships (
			college_id,
			group_id,
			user_id,
			membership_role
		)
		VALUES ($1, $2, $3, 'student')
	`, collegeB, groupB, studentB)
	if err != nil {
		t.Fatalf("create group B student membership: %v", err)
	}

	// Add faculty and student memberships to group A.
	_, err = tx.Exec(ctx, `
		INSERT INTO group_memberships (
			college_id,
			group_id,
			user_id,
			membership_role
		)
		VALUES
			($1, $2, $3, 'faculty'),
			($1, $2, $4, 'student'),
			($1, $2, $5, 'student')
	`, collegeA, groupA, facultyA, studentA, inactiveStudentA)
	if err != nil {
		t.Fatalf("create group memberships: %v", err)
	}

	service := NewService(tx)

	// 1. A faculty member of the group can create a subgroup.
	created, err := service.Create(
		ctx,
		collegeA,
		facultyA,
		"faculty",
		groupA,
		CreateInput{
			Name:        "Internship",
			Description: "Internship certificates",
		},
	)
	if err != nil {
		t.Fatalf("faculty create subgroup: %v", err)
	}

	if created.ID <= 0 {
		t.Fatalf("expected a valid subgroup ID, got %d", created.ID)
	}
	if created.CollegeID != collegeA {
		t.Errorf(
			"expected college ID %d, got %d",
			collegeA,
			created.CollegeID,
		)
	}
	if created.GroupID != groupA {
		t.Errorf(
			"expected group ID %d, got %d",
			groupA,
			created.GroupID,
		)
	}
	if created.Name != "Internship" {
		t.Errorf("expected subgroup name Internship, got %q", created.Name)
	}
	if !created.IsActive {
		t.Error("expected newly created subgroup to be active")
	}

	// Verify that subgroup creation notifies the group's active student.
	var notificationCount int
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM notifications
		WHERE recipient_user_id = $1
		  AND actor_user_id = $2
		  AND group_id = $3
		  AND subgroup_id = $4
		  AND type = 'subgroup_created'
	`, studentA, facultyA, groupA, created.ID).Scan(&notificationCount)
	if err != nil {
		t.Fatalf("count subgroup-created notifications: %v", err)
	}
	if notificationCount != 1 {
		t.Errorf(
			"expected 1 subgroup-created notification for student %d, got %d",
			studentA,
			notificationCount,
		)
	}

	// Students in another college/group must not receive this notification.
	var unrelatedNotificationCount int
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM notifications
		WHERE recipient_user_id = $1
		  AND group_id = $2
		  AND subgroup_id = $3
		  AND type = 'subgroup_created'
	`, studentB, groupA, created.ID).Scan(&unrelatedNotificationCount)
	if err != nil {
		t.Fatalf("count unrelated subgroup-created notifications: %v", err)
	}
	if unrelatedNotificationCount != 0 {
		t.Errorf(
			"expected 0 notifications for unrelated student %d, got %d",
			studentB,
			unrelatedNotificationCount,
		)
	}

	// Inactive students must not receive subgroup-created notifications.
	var inactiveNotificationCount int
	err = tx.QueryRow(ctx, `
    SELECT COUNT(*)
    FROM notifications
    WHERE recipient_user_id = $1
      AND group_id = $2
      AND subgroup_id = $3
      AND type = 'subgroup_created'
`, inactiveStudentA, groupA, created.ID).Scan(&inactiveNotificationCount)
	if err != nil {
		t.Fatalf("count inactive student notifications: %v", err)
	}
	if inactiveNotificationCount != 0 {
		t.Errorf(
			"expected 0 notifications for inactive student %d, got %d",
			inactiveStudentA,
			inactiveNotificationCount,
		)
	}

	// 2. An administrator in the same college can create a subgroup.
	adminCreated, err := service.Create(
		ctx,
		collegeA,
		adminA,
		"college_admin",
		groupA,
		CreateInput{
			Name:        "Events",
			Description: "Event certificates",
		},
	)
	if err != nil {
		t.Fatalf("admin create subgroup: %v", err)
	}

	if adminCreated.Name != "Events" {
		t.Errorf("expected subgroup name Events, got %q", adminCreated.Name)
	}

	// 3. Duplicate active subgroup names in the same group are rejected.
	_, err = service.Create(
		ctx,
		collegeA,
		facultyA,
		"faculty",
		groupA,
		CreateInput{
			Name:        "Internship",
			Description: "Duplicate name",
		},
	)
	if err != ErrSubgroupExists {
		t.Errorf("expected ErrSubgroupExists, got %v", err)
	}

	// Duplicate subgroup creation must not create another notification.
	err = tx.QueryRow(ctx, `
    SELECT COUNT(*)
    FROM notifications
    WHERE recipient_user_id = $1
      AND actor_user_id = $2
      AND group_id = $3
      AND subgroup_id = $4
      AND type = 'subgroup_created'
`, studentA, facultyA, groupA, created.ID).Scan(&notificationCount)
	if err != nil {
		t.Fatalf("count notifications after duplicate subgroup creation: %v", err)
	}
	if notificationCount != 1 {
		t.Errorf(
			"expected notification count to remain 1 after duplicate creation, got %d",
			notificationCount,
		)
	}

	// 4. A student cannot create a subgroup.
	_, err = service.Create(
		ctx,
		collegeA,
		studentA,
		"student",
		groupA,
		CreateInput{Name: "Student Created"},
	)
	if err != ErrForbidden {
		t.Errorf("expected ErrForbidden for student creation, got %v", err)
	}

	// 5. A faculty member without group membership cannot create one.
	_, err = service.Create(
		ctx,
		collegeA,
		outsiderFaculty,
		"faculty",
		groupA,
		CreateInput{Name: "Unauthorized"},
	)
	if err != ErrForbidden {
		t.Errorf(
			"expected ErrForbidden for non-member faculty creation, got %v",
			err,
		)
	}

	// 6. A group member can list the group's active subgroups.
	studentSubgroups, err := service.List(
		ctx,
		collegeA,
		studentA,
		"student",
		groupA,
	)
	if err != nil {
		t.Fatalf("student list subgroups: %v", err)
	}

	if len(studentSubgroups) != 2 {
		t.Fatalf(
			"expected 2 active subgroups for student, got %d",
			len(studentSubgroups),
		)
	}

	foundInternship := false
	foundEvents := false

	for _, subgroup := range studentSubgroups {
		switch subgroup.Name {
		case "Internship":
			foundInternship = true
		case "Events":
			foundEvents = true
		}
	}

	if !foundInternship || !foundEvents {
		t.Errorf(
			"expected Internship and Events in list; got %+v",
			studentSubgroups,
		)
	}

	// 7. A group faculty member can list subgroups.
	facultySubgroups, err := service.List(
		ctx,
		collegeA,
		facultyA,
		"faculty",
		groupA,
	)
	if err != nil {
		t.Fatalf("faculty list subgroups: %v", err)
	}
	if len(facultySubgroups) != 2 {
		t.Errorf(
			"expected 2 subgroups for faculty, got %d",
			len(facultySubgroups),
		)
	}

	// 8. A college administrator can list subgroups.
	adminSubgroups, err := service.List(
		ctx,
		collegeA,
		adminA,
		"college_admin",
		groupA,
	)
	if err != nil {
		t.Fatalf("admin list subgroups: %v", err)
	}
	if len(adminSubgroups) != 2 {
		t.Errorf(
			"expected 2 subgroups for admin, got %d",
			len(adminSubgroups),
		)
	}

	// 9. Faculty without membership cannot list the group's subgroups.
	_, err = service.List(
		ctx,
		collegeA,
		outsiderFaculty,
		"faculty",
		groupA,
	)
	if err != ErrForbidden {
		t.Errorf(
			"expected ErrForbidden for non-member faculty listing, got %v",
			err,
		)
	}

	// 10. College B cannot create a subgroup in college A's group.
	_, err = service.Create(
		ctx,
		collegeB,
		adminB,
		"college_admin",
		groupA,
		CreateInput{Name: "Cross College"},
	)
	if err != ErrGroupNotFound {
		t.Errorf(
			"expected ErrGroupNotFound for cross-college creation, got %v",
			err,
		)
	}

	// 11. College B cannot list college A's group subgroups.
	_, err = service.List(
		ctx,
		collegeB,
		adminB,
		"college_admin",
		groupA,
	)
	if err != ErrGroupNotFound {
		t.Errorf(
			"expected ErrGroupNotFound for cross-college listing, got %v",
			err,
		)
	}

	// 12. Invalid IDs must be rejected.
	_, err = service.Create(
		ctx,
		0,
		adminA,
		"college_admin",
		groupA,
		CreateInput{Name: "Invalid College"},
	)
	if err != ErrInvalidInput {
		t.Errorf("expected ErrInvalidInput for college ID 0, got %v", err)
	}

	_, err = service.List(
		ctx,
		collegeA,
		adminA,
		"college_admin",
		0,
	)
	if err != ErrInvalidInput {
		t.Errorf("expected ErrInvalidInput for group ID 0, got %v", err)
	}
}

func createSubgroupTestCollege(
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
		"Subgroup Integration College",
		uniqueCode(prefix),
	).Scan(&collegeID)
	if err != nil {
		t.Fatalf("create integration college: %v", err)
	}

	return collegeID
}

func createSubgroupTestUser(
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
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`,
		collegeID,
		"Subgroup Integration "+suffix,
		email,
		"integration-test-hash",
		role,
	).Scan(&userID)

	if err != nil {
		t.Fatalf("create integration user (%s): %v", role, err)
	}

	return userID
}

func uniqueCode(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
