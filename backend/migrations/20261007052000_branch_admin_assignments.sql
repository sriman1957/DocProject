-- +goose Up

BEGIN;

CREATE TABLE branch_admin_assignments (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    college_id BIGINT NOT NULL,
    branch_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,

    assigned_by BIGINT NOT NULL,

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (id, college_id),

    CONSTRAINT fk_branch_admin_assignments_branch
        FOREIGN KEY (branch_id, college_id)
        REFERENCES branches (id, college_id)
        ON DELETE CASCADE,

    CONSTRAINT fk_branch_admin_assignments_user
        FOREIGN KEY (user_id, college_id)
        REFERENCES users (id, college_id)
        ON DELETE CASCADE,

    CONSTRAINT fk_branch_admin_assignments_assigned_by
        FOREIGN KEY (assigned_by, college_id)
        REFERENCES users (id, college_id)
        ON DELETE RESTRICT,

    CONSTRAINT uq_branch_admin_assignments_branch_user
        UNIQUE (branch_id, user_id)
);

CREATE INDEX idx_branch_admin_assignments_college
    ON branch_admin_assignments (college_id);

CREATE INDEX idx_branch_admin_assignments_branch
    ON branch_admin_assignments (branch_id);

CREATE INDEX idx_branch_admin_assignments_user
    ON branch_admin_assignments (user_id);

COMMIT;

-- +goose Down

BEGIN;

DROP INDEX IF EXISTS idx_branch_admin_assignments_user;
DROP INDEX IF EXISTS idx_branch_admin_assignments_branch;
DROP INDEX IF EXISTS idx_branch_admin_assignments_college;

DROP TABLE IF EXISTS branch_admin_assignments;

COMMIT;
