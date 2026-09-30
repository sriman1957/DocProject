package accessperiods

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"docproject/backend/db"
	"docproject/backend/internal/config"

	"github.com/jackc/pgx/v5"
)

func TestAccessPeriodServiceIntegration(t *testing.T) {
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

	collegeA := createAccessPeriodTestCollege(
		t,
		ctx,
		tx,
		"APA",
	)

	collegeB := createAccessPeriodTestCollege(
		t,
		ctx,
		tx,
		"APB",
	)

	adminA := createAccessPeriodTestUser(
		t,
		ctx,
		tx,
		collegeA,
		"admin-a",
		"college_admin",
	)

	adminB := createAccessPeriodTestUser(
		t,
		ctx,
		tx,
		collegeB,
		"admin-b",
		"college_admin",
	)

	facultyAssigned := createAccessPeriodTestUser(
		t,
		ctx,
		tx,
		collegeA,
		"faculty-assigned",
		"faculty",
	)

	facultyUnassigned := createAccessPeriodTestUser(
		t,
		ctx,
		tx,
		collegeA,
		"faculty-unassigned",
		"faculty",
	)

	studentA := createAccessPeriodTestUser(
		t,
		ctx,
		tx,
		collegeA,
		"student-a",
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
		"Access Period Test Group",
		"Integration test group",
		adminA,
	).Scan(&groupA)

	if err != nil {
		t.Fatalf("create group A: %v", err)
	}

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
		"Access Period Isolation Group",
		"Tenant isolation test group",
		adminB,
	).Scan(&groupB)

	if err != nil {
		t.Fatalf("create group B: %v", err)
	}

	// Faculty must have an active faculty membership in the
	// parent group before they can be assigned to the subgroup.
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
		facultyAssigned,
	)

	if err != nil {
		t.Fatalf("create assigned faculty group membership: %v", err)
	}

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
		facultyUnassigned,
	)

	if err != nil {
		t.Fatalf("create unassigned faculty group membership: %v", err)
	}

	subgroupA := createAccessPeriodTestSubgroup(
		t,
		ctx,
		tx,
		collegeA,
		groupA,
		adminA,
		"Internship",
	)

	// Assign only one faculty member to the subgroup.
	_, err = tx.Exec(ctx, `
		INSERT INTO subgroup_faculty_assignments (
			college_id,
			subgroup_id,
			faculty_id,
			assigned_by
		)
		VALUES ($1, $2, $3, $4)
	`,
		collegeA,
		subgroupA,
		facultyAssigned,
		adminA,
	)

	if err != nil {
		t.Fatalf("create subgroup faculty assignment: %v", err)
	}

	service := NewService(tx)

	// 1. College admin can create an access period.
	first, err := service.Create(
		ctx,
		collegeA,
		adminA,
		"college_admin",
		groupA,
		subgroupA,
		CreateInput{
			StartsAt: "2026-10-01T10:00:00Z",
			EndsAt:   "2026-10-01T12:00:00Z",
		},
	)

	if err != nil {
		t.Fatalf("create access period: %v", err)
	}

	if first.ID <= 0 {
		t.Fatalf("expected valid access period ID, got %d", first.ID)
	}

	if first.CollegeID != collegeA {
		t.Errorf(
			"expected college ID %d, got %d",
			collegeA,
			first.CollegeID,
		)
	}

	if first.SubgroupID != subgroupA {
		t.Errorf(
			"expected subgroup ID %d, got %d",
			subgroupA,
			first.SubgroupID,
		)
	}

	if first.CreatedBy != adminA {
		t.Errorf(
			"expected created_by %d, got %d",
			adminA,
			first.CreatedBy,
		)
	}

	// 2. Assigned faculty can create an access period.
	assignedFacultyPeriod, err := service.Create(
		ctx,
		collegeA,
		facultyAssigned,
		"faculty",
		groupA,
		subgroupA,
		CreateInput{
			StartsAt: "2026-10-01T14:00:00Z",
			EndsAt:   "2026-10-01T16:00:00Z",
		},
	)

	if err != nil {
		t.Fatalf(
			"assigned faculty should be allowed to create access period: %v",
			err,
		)
	}

	if assignedFacultyPeriod.CreatedBy != facultyAssigned {
		t.Errorf(
			"expected created_by %d, got %d",
			facultyAssigned,
			assignedFacultyPeriod.CreatedBy,
		)
	}

	// 3. Unassigned faculty cannot create an access period.
	_, err = service.Create(
		ctx,
		collegeA,
		facultyUnassigned,
		"faculty",
		groupA,
		subgroupA,
		CreateInput{
			StartsAt: "2026-10-01T17:00:00Z",
			EndsAt:   "2026-10-01T18:00:00Z",
		},
	)

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf(
			"expected ErrForbidden for unassigned faculty, got %v",
			err,
		)
	}

	// 4. Students cannot create access periods.
	_, err = service.Create(
		ctx,
		collegeA,
		studentA,
		"student",
		groupA,
		subgroupA,
		CreateInput{
			StartsAt: "2026-10-01T17:00:00Z",
			EndsAt:   "2026-10-01T18:00:00Z",
		},
	)

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf(
			"expected ErrForbidden for student, got %v",
			err,
		)
	}

	// 5. An overlapping period must be rejected by PostgreSQL.
	//
	// PostgreSQL aborts the current transaction after a constraint
	// violation. Use a savepoint so the transaction remains usable
	// for the rest of the integration test.
	_, err = tx.Exec(
		ctx,
		"SAVEPOINT access_period_overlap_test",
	)
	if err != nil {
		t.Fatalf("create overlap savepoint: %v", err)
	}

	_, err = service.Create(
		ctx,
		collegeA,
		adminA,
		"college_admin",
		groupA,
		subgroupA,
		CreateInput{
			StartsAt: "2026-10-01T11:00:00Z",
			EndsAt:   "2026-10-01T13:00:00Z",
		},
	)

	if !errors.Is(err, ErrAccessPeriodOverlap) {
		t.Fatalf(
			"expected ErrAccessPeriodOverlap, got %v",
			err,
		)
	}

	_, err = tx.Exec(
		ctx,
		"ROLLBACK TO SAVEPOINT access_period_overlap_test",
	)
	if err != nil {
		t.Fatalf("rollback overlap savepoint: %v", err)
	}

	_, err = tx.Exec(
		ctx,
		"RELEASE SAVEPOINT access_period_overlap_test",
	)
	if err != nil {
		t.Fatalf("release overlap savepoint: %v", err)
	}

	// 6. Adjacent periods must be allowed.
	second, err := service.Create(
		ctx,
		collegeA,
		adminA,
		"college_admin",
		groupA,
		subgroupA,
		CreateInput{
			StartsAt: "2026-10-01T12:00:00Z",
			EndsAt:   "2026-10-01T14:00:00Z",
		},
	)

	if err != nil {
		t.Fatalf("create adjacent access period: %v", err)
	}

	if second.ID == first.ID {
		t.Fatal("expected a new access period record")
	}

	// 7. List must return all periods chronologically.
	periods, err := service.List(
		ctx,
		collegeA,
		adminA,
		"college_admin",
		groupA,
		subgroupA,
	)

	if err != nil {
		t.Fatalf("list access periods: %v", err)
	}

	if len(periods) != 3 {
		t.Fatalf(
			"expected 3 access periods, got %d",
			len(periods),
		)
	}

	if periods[0].ID != first.ID {
		t.Errorf(
			"expected first period ID %d, got %d",
			first.ID,
			periods[0].ID,
		)
	}

	if periods[1].ID != second.ID {
		t.Errorf(
			"expected second period ID %d, got %d",
			second.ID,
			periods[1].ID,
		)
	}

	// 8. Assigned faculty can list access periods.
	facultyPeriods, err := service.List(
		ctx,
		collegeA,
		facultyAssigned,
		"faculty",
		groupA,
		subgroupA,
	)

	if err != nil {
		t.Fatalf(
			"assigned faculty should be allowed to list periods: %v",
			err,
		)
	}

	if len(facultyPeriods) != 3 {
		t.Fatalf(
			"expected assigned faculty to see 3 periods, got %d",
			len(facultyPeriods),
		)
	}

	// 9. Unassigned faculty cannot list access periods.
	_, err = service.List(
		ctx,
		collegeA,
		facultyUnassigned,
		"faculty",
		groupA,
		subgroupA,
	)

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf(
			"expected ErrForbidden for unassigned faculty list, got %v",
			err,
		)
	}

	// 10. Students cannot list access periods.
	_, err = service.List(
		ctx,
		collegeA,
		studentA,
		"student",
		groupA,
		subgroupA,
	)

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf(
			"expected ErrForbidden for student list, got %v",
			err,
		)
	}

	// 11. Current must return the first period.
	current, err := service.Current(
		ctx,
		collegeA,
		groupA,
		subgroupA,
		"2026-10-01T10:30:00Z",
	)

	if err != nil {
		t.Fatalf("get current first period: %v", err)
	}

	if current.ID != first.ID {
		t.Errorf(
			"expected current period %d, got %d",
			first.ID,
			current.ID,
		)
	}

	// 12. At exactly 12:00, the first period is closed
	// and the second period becomes active.
	current, err = service.Current(
		ctx,
		collegeA,
		groupA,
		subgroupA,
		"2026-10-01T12:00:00Z",
	)

	if err != nil {
		t.Fatalf("get current second period: %v", err)
	}

	if current.ID != second.ID {
		t.Errorf(
			"expected current period %d at boundary, got %d",
			second.ID,
			current.ID,
		)
	}

	// 13. After the second period closes, the third period
	// starts at 14:00. At exactly 14:00 it becomes current.
	current, err = service.Current(
		ctx,
		collegeA,
		groupA,
		subgroupA,
		"2026-10-01T14:00:00Z",
	)

	if err != nil {
		t.Fatalf("get current third period: %v", err)
	}

	if current.ID != assignedFacultyPeriod.ID {
		t.Errorf(
			"expected current period %d at 14:00, got %d",
			assignedFacultyPeriod.ID,
			current.ID,
		)
	}

	// 14. After all periods close, no current period exists.
	_, err = service.Current(
		ctx,
		collegeA,
		groupA,
		subgroupA,
		"2026-10-01T16:00:00Z",
	)

	if err != ErrAccessPeriodNotFound {
		t.Errorf(
			"expected ErrAccessPeriodNotFound after closing period, got %v",
			err,
		)
	}

	// 15. College B must not access college A's subgroup.
	_, err = service.List(
		ctx,
		collegeB,
		adminB,
		"college_admin",
		groupB,
		subgroupA,
	)

	if err != ErrSubgroupNotFound {
		t.Errorf(
			"expected ErrSubgroupNotFound for cross-college list, got %v",
			err,
		)
	}

	// 16. College B must not create a period for college A's subgroup.
	_, err = service.Create(
		ctx,
		collegeB,
		adminB,
		"college_admin",
		groupB,
		subgroupA,
		CreateInput{
			StartsAt: "2026-10-02T10:00:00Z",
			EndsAt:   "2026-10-02T12:00:00Z",
		},
	)

	if err != ErrSubgroupNotFound {
		t.Errorf(
			"expected ErrSubgroupNotFound for cross-college create, got %v",
			err,
		)
	}

	// 17. The subgroup must belong to the requested group.
	_, err = service.Create(
		ctx,
		collegeA,
		adminA,
		"college_admin",
		groupB,
		subgroupA,
		CreateInput{
			StartsAt: "2026-10-02T10:00:00Z",
			EndsAt:   "2026-10-02T12:00:00Z",
		},
	)

	if err != ErrSubgroupNotFound {
		t.Errorf(
			"expected ErrSubgroupNotFound for wrong group, got %v",
			err,
		)
	}

	// 18. Revoking a faculty assignment removes access.
	_, err = tx.Exec(ctx, `
		UPDATE subgroup_faculty_assignments
		SET
			revoked_by = $1,
			revoked_at = NOW()
		WHERE college_id = $2
		  AND subgroup_id = $3
		  AND faculty_id = $4
		  AND revoked_at IS NULL
	`,
		adminA,
		collegeA,
		subgroupA,
		facultyAssigned,
	)

	if err != nil {
		t.Fatalf("revoke faculty assignment: %v", err)
	}

	_, err = service.Create(
		ctx,
		collegeA,
		facultyAssigned,
		"faculty",
		groupA,
		subgroupA,
		CreateInput{
			StartsAt: "2026-10-02T10:00:00Z",
			EndsAt:   "2026-10-02T12:00:00Z",
		},
	)

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf(
			"expected ErrForbidden after faculty assignment revocation, got %v",
			err,
		)
	}

	// 19. An inactive subgroup must reject new access periods.
	_, err = tx.Exec(
		ctx,
		`
		UPDATE subgroups
		SET
			is_active = FALSE,
			deleted_at = NOW(),
			deleted_by = $1
		WHERE id = $2
		  AND college_id = $3
		`,
		adminA,
		subgroupA,
		collegeA,
	)

	if err != nil {
		t.Fatalf("deactivate subgroup: %v", err)
	}

	_, err = service.Create(
		ctx,
		collegeA,
		adminA,
		"college_admin",
		groupA,
		subgroupA,
		CreateInput{
			StartsAt: "2026-10-02T10:00:00Z",
			EndsAt:   "2026-10-02T12:00:00Z",
		},
	)

	if err != ErrSubgroupNotFound {
		t.Errorf(
			"expected ErrSubgroupNotFound for inactive subgroup, got %v",
			err,
		)
	}
}

func createAccessPeriodTestCollege(
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
		"Access Period Integration College",
		accessPeriodUniqueCode(prefix),
	).Scan(&collegeID)

	if err != nil {
		t.Fatalf("create access period college: %v", err)
	}

	return collegeID
}

func createAccessPeriodTestUser(
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
		accessPeriodUniqueCode("user"),
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
		"Access Period Integration "+suffix,
		email,
		"integration-test-hash",
		role,
	).Scan(&userID)

	if err != nil {
		t.Fatalf(
			"create access period user (%s): %v",
			role,
			err,
		)
	}

	return userID
}

func createAccessPeriodTestSubgroup(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	collegeID int64,
	groupID int64,
	createdBy int64,
	name string,
) int64 {
	t.Helper()

	var subgroupID int64

	err := tx.QueryRow(ctx, `
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
		name,
		"Access period integration subgroup",
		createdBy,
	).Scan(&subgroupID)

	if err != nil {
		t.Fatalf("create access period subgroup: %v", err)
	}

	return subgroupID
}

func accessPeriodUniqueCode(prefix string) string {
	return fmt.Sprintf(
		"%s_%d",
		prefix,
		time.Now().UnixNano(),
	)
}
