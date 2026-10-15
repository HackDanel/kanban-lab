-- name: CreateBoard :one
INSERT INTO boards (project_id, name)
VALUES (@project_id, @name)
RETURNING *;

-- name: GetBoard :one
SELECT * FROM boards WHERE id = @id;

-- name: LockBoard :one
SELECT * FROM boards WHERE id = @id FOR UPDATE;

-- name: ListBoards :many
SELECT * FROM boards WHERE project_id = @project_id ORDER BY created_at, id;

-- name: RenameBoard :one
UPDATE boards SET name = @name, updated_at = now() WHERE id = @id RETURNING *;

-- name: DeleteBoard :execrows
DELETE FROM boards WHERE id = @id;

-- name: CreateColumn :one
INSERT INTO board_columns (board_id, name, position, wip_limit)
VALUES (@board_id, @name, @position, @wip_limit)
RETURNING *;

-- name: GetColumn :one
SELECT * FROM board_columns WHERE id = @id;

-- name: LockColumn :one
SELECT * FROM board_columns WHERE id = @id FOR UPDATE;

-- name: ListColumns :many
SELECT col.*,
       (SELECT count(*) FROM cards c WHERE c.column_id = col.id)::integer AS card_count
FROM board_columns col
WHERE col.board_id = @board_id
ORDER BY col.position, col.id;

-- name: RenameColumn :exec
UPDATE board_columns SET name = @name WHERE id = @id;

-- name: SetColumnWipLimit :exec
UPDATE board_columns SET wip_limit = sqlc.narg('wip_limit') WHERE id = @id;

-- name: SetColumnPosition :exec
UPDATE board_columns SET position = @position WHERE id = @id;

-- name: DeleteColumn :execrows
DELETE FROM board_columns WHERE id = @id;

-- name: MaxColumnPosition :one
SELECT COALESCE(max(position), 0)::double precision FROM board_columns WHERE board_id = @board_id;

-- name: ColumnPositions :many
SELECT id, position FROM board_columns
WHERE board_id = @board_id AND id <> @exclude_id
ORDER BY position, id;

-- name: RenumberColumns :exec
UPDATE board_columns col
SET position = ranked.rn * 1024
FROM (SELECT b.id, row_number() OVER (ORDER BY b.position, b.id) AS rn
      FROM board_columns b
      WHERE b.board_id = @board_id) ranked
WHERE col.id = ranked.id;
