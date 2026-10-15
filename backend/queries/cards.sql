-- name: CreateCard :one
INSERT INTO cards (project_id, board_id, column_id, number, title, description, priority, position,
                   assignee_id, reporter_id, due_date)
VALUES (@project_id, @board_id, @column_id, @number, @title, @description, @priority, @position,
        @assignee_id, @reporter_id, @due_date)
RETURNING id;

-- name: GetCard :one
SELECT * FROM card_views WHERE id = @id;

-- name: LockCard :one
SELECT id FROM cards WHERE id = @id FOR UPDATE;

-- name: ListCards :many
SELECT * FROM card_views c
WHERE c.board_id = @board_id
  AND (sqlc.narg('column_id')::uuid IS NULL OR c.column_id = sqlc.narg('column_id'))
  AND (sqlc.narg('assignee_id')::uuid IS NULL OR c.assignee_id = sqlc.narg('assignee_id'))
  AND (sqlc.narg('priority')::text IS NULL OR c.priority = sqlc.narg('priority'))
  AND (sqlc.narg('pattern')::text IS NULL
       OR c.title ILIKE sqlc.narg('pattern')
       OR c.description ILIKE sqlc.narg('pattern'))
  AND (COALESCE(cardinality(@label_ids::uuid[]), 0) = 0
       OR (SELECT count(*) FROM card_labels cl
           WHERE cl.card_id = c.id AND cl.label_id = ANY (@label_ids::uuid[]))
          = cardinality(@label_ids::uuid[]))
ORDER BY c.column_id, c.position, c.id
LIMIT @page_limit OFFSET @page_offset;

-- name: CountCards :one
SELECT count(*) FROM cards c
WHERE c.board_id = @board_id
  AND (sqlc.narg('column_id')::uuid IS NULL OR c.column_id = sqlc.narg('column_id'))
  AND (sqlc.narg('assignee_id')::uuid IS NULL OR c.assignee_id = sqlc.narg('assignee_id'))
  AND (sqlc.narg('priority')::text IS NULL OR c.priority = sqlc.narg('priority'))
  AND (sqlc.narg('pattern')::text IS NULL
       OR c.title ILIKE sqlc.narg('pattern')
       OR c.description ILIKE sqlc.narg('pattern'))
  AND (COALESCE(cardinality(@label_ids::uuid[]), 0) = 0
       OR (SELECT count(*) FROM card_labels cl
           WHERE cl.card_id = c.id AND cl.label_id = ANY (@label_ids::uuid[]))
          = cardinality(@label_ids::uuid[]));

-- name: UpdateCardFields :exec
UPDATE cards
SET title       = @title,
    description = @description,
    priority    = @priority,
    assignee_id = @assignee_id,
    due_date    = @due_date,
    updated_at  = now()
WHERE id = @id;

-- name: SetCardPlace :exec
UPDATE cards SET column_id = @column_id, position = @position, updated_at = now() WHERE id = @id;

-- name: DeleteCard :execrows
DELETE FROM cards WHERE id = @id;

-- name: CountCardsInColumn :one
SELECT count(*) FROM cards WHERE column_id = @column_id;

-- name: MaxCardPosition :one
SELECT COALESCE(max(position), 0)::double precision FROM cards WHERE column_id = @column_id;

-- name: CardPositions :many
SELECT id, position FROM cards
WHERE column_id = @column_id AND id <> @exclude_id
ORDER BY position, id;

-- name: RenumberCards :exec
UPDATE cards c
SET position = ranked.rn * 1024
FROM (SELECT k.id, row_number() OVER (ORDER BY k.position, k.id) AS rn
      FROM cards k
      WHERE k.column_id = @column_id) ranked
WHERE c.id = ranked.id;

-- name: ClearCardLabels :exec
DELETE FROM card_labels WHERE card_id = @card_id;

-- name: AddCardLabels :exec
INSERT INTO card_labels (card_id, label_id)
SELECT @card_id, unnest(@label_ids::uuid[]);

-- name: InsertActivity :exec
INSERT INTO card_activity (card_id, actor_id, action, field, old_value, new_value)
VALUES (@card_id, @actor_id, @action, @field, @old_value, @new_value);

-- name: ListActivity :many
SELECT * FROM card_activity
WHERE card_id = @card_id
ORDER BY created_at DESC, id
LIMIT @page_limit OFFSET @page_offset;

-- name: CreateComment :one
INSERT INTO comments (card_id, author_id, body)
VALUES (@card_id, @author_id, @body)
RETURNING *;

-- name: GetComment :one
SELECT * FROM comments WHERE id = @id;

-- name: ListComments :many
SELECT * FROM comments WHERE card_id = @card_id ORDER BY created_at, id;

-- name: UpdateComment :one
UPDATE comments SET body = @body, updated_at = now() WHERE id = @id RETURNING *;

-- name: DeleteComment :execrows
DELETE FROM comments WHERE id = @id;

-- name: CreateAttachment :one
INSERT INTO attachments (card_id, uploader_id, filename, content_type, size, storage_key)
VALUES (@card_id, @uploader_id, @filename, @content_type, @size, @storage_key)
RETURNING *;

-- name: GetAttachment :one
SELECT * FROM attachments WHERE id = @id;

-- name: ListAttachments :many
SELECT * FROM attachments WHERE card_id = @card_id ORDER BY created_at, id;

-- name: DeleteAttachment :execrows
DELETE FROM attachments WHERE id = @id;

-- name: ListAttachmentKeysForProject :many
SELECT a.storage_key FROM attachments a JOIN cards c ON c.id = a.card_id WHERE c.project_id = @project_id;

-- name: ListAttachmentKeysForBoard :many
SELECT a.storage_key FROM attachments a JOIN cards c ON c.id = a.card_id WHERE c.board_id = @board_id;

-- name: ListAttachmentKeysForCard :many
SELECT storage_key FROM attachments WHERE card_id = @card_id;
