-- +goose Up

CREATE TABLE subgroup_faculty_assignments (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    college_id BIGINT NOT NULL,
    subgroup_id BIGINT NOT NULL,
    faculty_id BIGINT NOT NULL,

    assigned_by BIGINT NOT NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    revoked_by BIGINT,
    revoked_at TIMESTAMPTZ,

    CONSTRAINT subgroup_faculty_assignments_subgroup_college_fkey
        FOREIGN KEY (subgroup_id, college_id)
        REFERENCES subgroups (id, college_id)
        ON DELETE RESTRICT,

    CONSTRAINT subgroup_faculty_assignments_faculty_college_fkey
        FOREIGN KEY (faculty_id, college_id)
        REFERENCES users (id, college_id)
        ON DELETE RESTRICT,

    CONSTRAINT subgroup_faculty_assignments_assigned_by_college_fkey
        FOREIGN KEY (assigned_by, college_id)
        REFERENCES users (id, college_id)
        ON DELETE RESTRICT,

    CONSTRAINT subgroup_faculty_assignments_revoked_by_college_fkey
        FOREIGN KEY (revoked_by, college_id)
        REFERENCES users (id, college_id)
        ON DELETE RESTRICT,

    CONSTRAINT subgroup_faculty_assignments_revocation_check
        CHECK (
            (revoked_at IS NULL AND revoked_by IS NULL)
            OR
            (revoked_at IS NOT NULL AND revoked_by IS NOT NULL)
        )
);

-- Only one active assignment per faculty member per subgroup.
CREATE UNIQUE INDEX uq_subgroup_faculty_assignments_active
    ON subgroup_faculty_assignments (subgroup_id, faculty_id)
    WHERE revoked_at IS NULL;

-- Efficiently list assignments for a subgroup.
CREATE INDEX idx_subgroup_faculty_assignments_subgroup
    ON subgroup_faculty_assignments (subgroup_id, assigned_at);

-- Efficiently find active subgroup assignments for a faculty member.
CREATE INDEX idx_subgroup_faculty_assignments_faculty_active
    ON subgroup_faculty_assignments (faculty_id, subgroup_id)
    WHERE revoked_at IS NULL;


-- +goose Down

DROP TABLE IF EXISTS subgroup_faculty_assignments;
