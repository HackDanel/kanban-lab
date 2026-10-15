-- name: CreateProject :one
INSERT INTO projects (key, name, description, owner_id)
VALUES (@key, @name, @description, @owner_id)
RETURNING *;

-- name: GetProject :one
SELECT * FROM projects WHERE id = @id;

-- name: ListAllProjects :many
SELECT * FROM projects ORDER BY name, id;

-- name: ListProjectsForMember :many
SELECT p.*
FROM projects p
JOIN project_members m ON m.project_id = p.id
WHERE m.user_id = @user_id
ORDER BY p.name, p.id;

-- name: UpdateProject :one
UPDATE projects
SET name        = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    updated_at  = now()
WHERE id = @id
RETURNING *;

-- name: DeleteProject :execrows
DELETE FROM projects WHERE id = @id;

-- name: NextCardNumber :one
UPDATE projects SET card_seq = card_seq + 1 WHERE id = @id RETURNING card_seq;

-- name: IsProjectMember :one
SELECT EXISTS (SELECT 1 FROM project_members WHERE project_id = @project_id AND user_id = @user_id);

-- name: AddProjectMember :exec
INSERT INTO project_members (project_id, user_id)
VALUES (@project_id, @user_id)
ON CONFLICT DO NOTHING;

-- name: RemoveProjectMember :execrows
DELETE FROM project_members WHERE project_id = @project_id AND user_id = @user_id;

-- name: ListProjectMembers :many
SELECT u.*
FROM users u
JOIN project_members m ON m.user_id = u.id
WHERE m.project_id = @project_id
ORDER BY u.name, u.id;

-- name: UnassignMemberCards :exec
UPDATE cards SET assignee_id = NULL, updated_at = now()
WHERE project_id = @project_id AND assignee_id = @user_id;

-- name: CreateLabel :one
INSERT INTO labels (project_id, name, color)
VALUES (@project_id, @name, @color)
RETURNING *;

-- name: GetLabel :one
SELECT * FROM labels WHERE id = @id;

-- name: ListLabels :many
SELECT * FROM labels WHERE project_id = @project_id ORDER BY name, id;

-- name: UpdateLabel :one
UPDATE labels
SET name  = COALESCE(sqlc.narg('name'), name),
    color = COALESCE(sqlc.narg('color'), color)
WHERE id = @id
RETURNING *;

-- name: DeleteLabel :execrows
DELETE FROM labels WHERE id = @id;

-- name: CountLabelsInProject :one
SELECT count(*) FROM labels WHERE project_id = @project_id AND id = ANY (@ids::uuid[]);
