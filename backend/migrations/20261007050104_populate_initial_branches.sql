-- +goose Up

BEGIN;

-- =========================================================
-- 1. Create initial branches for existing colleges
-- =========================================================

INSERT INTO branches (
    college_id,
    name,
    code,
    description
)
VALUES
    (
        1,
        'Computer Science and Engineering',
        'CSE',
        'Computer Science and Engineering branch'
    ),
    (
        2,
        'Information Technology',
        'IT',
        'Information Technology branch'
    );

-- =========================================================
-- 2. Associate existing batch groups with their branches
-- =========================================================

UPDATE groups
SET branch_id = (
    SELECT b.id
    FROM branches b
    WHERE b.college_id = groups.college_id
      AND b.code = 'CSE'
)
WHERE college_id = 1
  AND name = 'CSE 2024-2028';

UPDATE groups
SET branch_id = (
    SELECT b.id
    FROM branches b
    WHERE b.college_id = groups.college_id
      AND b.code = 'IT'
)
WHERE college_id = 2
  AND name = 'IT 2024-2028';

-- =========================================================
-- 3. Every existing batch must belong to a branch
-- =========================================================

ALTER TABLE groups
    ALTER COLUMN branch_id SET NOT NULL;

COMMIT;


-- +goose Down

BEGIN;

-- Allow NULL temporarily so existing branch associations
-- can be removed safely.

ALTER TABLE groups
    ALTER COLUMN branch_id DROP NOT NULL;

-- Remove the branch association from the existing groups.

UPDATE groups
SET branch_id = NULL
WHERE college_id = 1
  AND name = 'CSE 2024-2028';

UPDATE groups
SET branch_id = NULL
WHERE college_id = 2
  AND name = 'IT 2024-2028';

-- Remove only the branches created by this migration.

DELETE FROM branches
WHERE (college_id = 1 AND code = 'CSE')
   OR (college_id = 2 AND code = 'IT');

COMMIT;
