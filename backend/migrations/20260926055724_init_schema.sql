-- +goose Up

BEGIN;

-- =========================================================
-- 1. Colleges
-- =========================================================

CREATE TABLE colleges (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    code VARCHAR(50) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =========================================================
-- 2. Users
-- =========================================================

CREATE TABLE users (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    college_id BIGINT NOT NULL
        REFERENCES colleges(id) ON DELETE RESTRICT,

    full_name VARCHAR(200) NOT NULL,
    email VARCHAR(254) NOT NULL,
    password_hash TEXT NOT NULL,

    -- Public faculty identifier, e.g. T014.
    -- NULL for users who do not have a faculty ID.
    faculty_code VARCHAR(50),

    role VARCHAR(20) NOT NULL
        CHECK (role IN ('student', 'faculty', 'college_admin')),

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (college_id, email),
    UNIQUE (id, college_id),
    UNIQUE (college_id, faculty_code)
);

CREATE INDEX idx_users_college_role
    ON users (college_id, role);

-- =========================================================
-- 3. Main groups
-- Example: IT 2024-2028
-- =========================================================

CREATE TABLE groups (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    college_id BIGINT NOT NULL
        REFERENCES colleges(id) ON DELETE RESTRICT,

    name VARCHAR(200) NOT NULL,
    description TEXT,

    created_by BIGINT NOT NULL,

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (id, college_id),

    FOREIGN KEY (created_by, college_id)
        REFERENCES users (id, college_id)
        ON DELETE RESTRICT
);

CREATE INDEX idx_groups_college
    ON groups (college_id);

-- =========================================================
-- 4. Main group memberships
-- Faculty permissions are inherited from this table.
-- Every faculty member in a group can manage its subgroups.
-- =========================================================

CREATE TABLE group_memberships (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    college_id BIGINT NOT NULL,
    group_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,

    membership_role VARCHAR(20) NOT NULL
        CHECK (membership_role IN ('student', 'faculty')),

    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    FOREIGN KEY (group_id, college_id)
        REFERENCES groups (id, college_id)
        ON DELETE RESTRICT,

    FOREIGN KEY (user_id, college_id)
        REFERENCES users (id, college_id)
        ON DELETE RESTRICT,

    UNIQUE (group_id, user_id),
    UNIQUE (group_id, user_id, college_id)
);

CREATE INDEX idx_group_memberships_user
    ON group_memberships (user_id, group_id);

CREATE INDEX idx_group_memberships_group_role
    ON group_memberships (group_id, membership_role, user_id);

-- =========================================================
-- 5. Subgroups
-- Example: Global Certificates, Internship, Events
--
-- Deletion is archival. Documents remain linked.
-- =========================================================

CREATE TABLE subgroups (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    college_id BIGINT NOT NULL,
    group_id BIGINT NOT NULL,

    name VARCHAR(200) NOT NULL,
    description TEXT,

    created_by BIGINT NOT NULL,

    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    deleted_at TIMESTAMPTZ,
    deleted_by BIGINT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (id, college_id),

    FOREIGN KEY (group_id, college_id)
        REFERENCES groups (id, college_id)
        ON DELETE RESTRICT,

    FOREIGN KEY (created_by, college_id)
        REFERENCES users (id, college_id)
        ON DELETE RESTRICT,

    FOREIGN KEY (deleted_by, college_id)
        REFERENCES users (id, college_id)
        ON DELETE RESTRICT,

    CHECK (
        (is_active = TRUE AND deleted_at IS NULL AND deleted_by IS NULL)
        OR
        (is_active = FALSE AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL)
    )
);

-- Active subgroup names must be unique within a main group.
-- Archived subgroup names may be reused.
CREATE UNIQUE INDEX uq_subgroups_active_name
    ON subgroups (group_id, name)
    WHERE is_active = TRUE;

CREATE INDEX idx_subgroups_group
    ON subgroups (group_id, is_active);

-- =========================================================
-- 6. Subgroup access periods
-- Each reopening creates a new period.
-- Times include timezone information.
-- =========================================================

CREATE TABLE subgroup_access_periods (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    subgroup_id BIGINT NOT NULL,
    college_id BIGINT NOT NULL,

    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,

    created_by BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CHECK (ends_at > starts_at),

    FOREIGN KEY (subgroup_id, college_id)
        REFERENCES subgroups (id, college_id)
        ON DELETE RESTRICT,

    FOREIGN KEY (created_by, college_id)
        REFERENCES users (id, college_id)
        ON DELETE RESTRICT
);

CREATE INDEX idx_access_periods_subgroup_time
    ON subgroup_access_periods (subgroup_id, starts_at, ends_at);

-- =========================================================
-- 7. Documents
--
-- Personal Vault document:
--   subgroup_id IS NULL
--
-- Subgroup document:
--   subgroup_id IS NOT NULL
--
-- Sharing from the vault creates a new document record
-- and a separate physical file in subgroup storage.
-- =========================================================

CREATE TABLE documents (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    college_id BIGINT NOT NULL,
    owner_id BIGINT NOT NULL,
    subgroup_id BIGINT,

    original_filename VARCHAR(255) NOT NULL,
    storage_key TEXT NOT NULL UNIQUE,

    mime_type VARCHAR(127) NOT NULL
        CHECK (
            mime_type IN (
                'application/pdf',
                'application/vnd.openxmlformats-officedocument.wordprocessingml.document'
            )
        ),

    file_size_bytes BIGINT NOT NULL
        CHECK (file_size_bytes > 0),

    sha256 CHAR(64) NOT NULL,

    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Source vault document for a physical copy.
    copied_from_document_id BIGINT,

    -- Soft deletion preserves metadata and file history.
    deleted_at TIMESTAMPTZ,
    deleted_by BIGINT,

    FOREIGN KEY (owner_id, college_id)
        REFERENCES users (id, college_id)
        ON DELETE RESTRICT,

    FOREIGN KEY (subgroup_id, college_id)
        REFERENCES subgroups (id, college_id)
        ON DELETE RESTRICT,

    FOREIGN KEY (copied_from_document_id)
        REFERENCES documents (id)
        ON DELETE RESTRICT,

    FOREIGN KEY (deleted_by, college_id)
        REFERENCES users (id, college_id)
        ON DELETE RESTRICT,

    CHECK (
        (subgroup_id IS NULL AND copied_from_document_id IS NULL)
        OR subgroup_id IS NOT NULL
    ),

    CHECK (
        (deleted_at IS NULL AND deleted_by IS NULL)
        OR
        (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)
    )
);

CREATE INDEX idx_documents_owner
    ON documents (owner_id, uploaded_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_documents_subgroup
    ON documents (subgroup_id, uploaded_at DESC)
    WHERE subgroup_id IS NOT NULL
      AND deleted_at IS NULL;

CREATE INDEX idx_documents_copied_from
    ON documents (copied_from_document_id)
    WHERE copied_from_document_id IS NOT NULL;

-- =========================================================
-- 8. Document audit logs
-- Records uploads, copies, and deletions.
-- =========================================================

CREATE TABLE document_audit_logs (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    document_id BIGINT NOT NULL
        REFERENCES documents(id) ON DELETE RESTRICT,

    actor_user_id BIGINT NOT NULL
        REFERENCES users(id) ON DELETE RESTRICT,

    action VARCHAR(50) NOT NULL
        CHECK (
            action IN (
                'document_uploaded',
                'document_copied',
                'document_deleted'
            )
        ),

    filename VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_document_audit_document
    ON document_audit_logs (document_id, created_at DESC);

CREATE INDEX idx_document_audit_actor
    ON document_audit_logs (actor_user_id, created_at DESC);

-- =========================================================
-- 9. In-app notifications
-- =========================================================

CREATE TABLE notifications (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    recipient_user_id BIGINT NOT NULL
        REFERENCES users(id) ON DELETE RESTRICT,

    actor_user_id BIGINT NOT NULL
        REFERENCES users(id) ON DELETE RESTRICT,

    group_id BIGINT NOT NULL
        REFERENCES groups(id) ON DELETE RESTRICT,

    subgroup_id BIGINT
        REFERENCES subgroups(id) ON DELETE RESTRICT,

    document_id BIGINT
        REFERENCES documents(id) ON DELETE RESTRICT,

    type VARCHAR(50) NOT NULL
        CHECK (
            type IN (
                'subgroup_created',
                'subgroup_updated',
                'subgroup_deleted',
                'access_period_opened',
                'document_deleted'
            )
        ),

    message TEXT NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_recipient
    ON notifications (recipient_user_id, created_at DESC);

CREATE INDEX idx_notifications_unread
    ON notifications (recipient_user_id, created_at DESC)
    WHERE is_read = FALSE;

COMMIT;


-- +goose Down

BEGIN;

DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS document_audit_logs;
DROP TABLE IF EXISTS documents;
DROP TABLE IF EXISTS subgroup_access_periods;
DROP TABLE IF EXISTS subgroups;
DROP TABLE IF EXISTS group_memberships;
DROP TABLE IF EXISTS groups;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS colleges;

COMMIT;
