-- +goose Up

-- Enable GiST equality support for bigint columns.
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- Prevent overlapping access periods for the same subgroup.
-- Adjacent periods are allowed: [10:00, 11:00) and [11:00, 12:00).
ALTER TABLE subgroup_access_periods
ADD CONSTRAINT subgroup_access_periods_no_overlap
EXCLUDE USING gist (
    subgroup_id WITH =,
    tstzrange(starts_at, ends_at, '[)') WITH &&
);


-- +goose Down

ALTER TABLE subgroup_access_periods
DROP CONSTRAINT IF EXISTS subgroup_access_periods_no_overlap;
