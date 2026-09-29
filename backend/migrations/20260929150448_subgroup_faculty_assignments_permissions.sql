-- +goose Up
GRANT SELECT, INSERT, UPDATE
ON TABLE subgroup_faculty_assignments
TO docproject_app;

GRANT USAGE, SELECT
ON SEQUENCE subgroup_faculty_assignments_id_seq
TO docproject_app;


-- +goose Down
REVOKE USAGE, SELECT
ON SEQUENCE subgroup_faculty_assignments_id_seq
FROM docproject_app;

REVOKE SELECT, INSERT, UPDATE
ON TABLE subgroup_faculty_assignments
FROM docproject_app;
