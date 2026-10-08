package branchadmins

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

var (
	ErrNotBranchAdmin = errors.New("user is not an active branch admin")
)

type AuthorizationQuerier interface {
	QueryRow(
		ctx context.Context,
		sql string,
		args ...any,
	) pgx.Row
}

// IsBranchAdmin verifies that the user is an active Branch Admin
// for the specified branch within the specified college.
func IsBranchAdmin(
	ctx context.Context,
	db AuthorizationQuerier,
	collegeID int64,
	userID int64,
	branchID int64,
) (bool, error) {
	if collegeID <= 0 ||
		userID <= 0 ||
		branchID <= 0 {
		return false, nil
	}

	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM branch_admin_assignments
			WHERE college_id = $1
			  AND branch_id = $2
			  AND user_id = $3
			  AND is_active = TRUE
		)
	`

	var exists bool

	if err := db.QueryRow(
		ctx,
		query,
		collegeID,
		branchID,
		userID,
	).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}
