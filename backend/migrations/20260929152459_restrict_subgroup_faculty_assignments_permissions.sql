-- +goose Up
REVOKE DELETE
ON TABLE subgroup_faculty_assignments
FROM docproject_app;

-- +goose Down
GRANT DELETE
ON TABLE subgroup_faculty_assignments
TO docproject_app;
