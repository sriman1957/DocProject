-- +goose Up

BEGIN;

-- =========================================================
-- 1. Branches
-- =========================================================
-- A college can have multiple branches/departments.
-- Example:
--   College A
--     ├── CSE
--     ├── IT
--     ├── ECE
--     └── EEE
--
-- Branches are created and managed by the College Admin.
-- =========================================================

CREATE TABLE branches (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    college_id BIGINT NOT NULL
        REFERENCES colleges(id) ON DELETE RESTRICT,

    name VARCHAR(200) NOT NULL,
    code VARCHAR(50) NOT NULL,

    description TEXT,

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (id, college_id),
    UNIQUE (college_id, code),
    UNIQUE (college_id, name)
);

CREATE INDEX idx_branches_college
    ON branches (college_id);

-- =========================================================
-- 2. Add branch relationship to groups
-- =========================================================
-- Existing groups represent batch groups.
--
-- Example:
--   Branch: CSE
--      └── Group: CSE 2024-2028
--
-- We initially allow NULL so existing data can be migrated
-- safely before making the relationship mandatory.
-- =========================================================

ALTER TABLE groups
    ADD COLUMN branch_id BIGINT;

-- =========================================================
-- 3. Temporary foreign key
-- =========================================================

ALTER TABLE groups
    ADD CONSTRAINT fk_groups_branch
    FOREIGN KEY (branch_id, college_id)
    REFERENCES branches (id, college_id)
    ON DELETE RESTRICT;

CREATE INDEX idx_groups_branch
    ON groups (branch_id);

COMMIT;


-- +goose Down

BEGIN;

DROP INDEX IF EXISTS idx_groups_branch;

ALTER TABLE groups
    DROP CONSTRAINT IF EXISTS fk_groups_branch;

ALTER TABLE groups
    DROP COLUMN IF EXISTS branch_id;

DROP INDEX IF EXISTS idx_branches_college;

DROP TABLE IF EXISTS branches;

COMMIT;
