-- name: CreateUser :one
INSERT INTO users (email, name, password_hash, role)
VALUES (@email, @name, @password_hash, @role)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = @id;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = @email;

-- name: CountUsers :one
SELECT count(*) FROM users;

-- name: ListUsers :many
SELECT * FROM users
WHERE sqlc.narg('pattern')::text IS NULL
   OR name ILIKE sqlc.narg('pattern')
   OR email::text ILIKE sqlc.narg('pattern')
ORDER BY name, id
LIMIT @page_limit OFFSET @page_offset;

-- name: CountUsersMatching :one
SELECT count(*) FROM users
WHERE sqlc.narg('pattern')::text IS NULL
   OR name ILIKE sqlc.narg('pattern')
   OR email::text ILIKE sqlc.narg('pattern');

-- name: SetUserRole :one
UPDATE users SET role = @role WHERE id = @id RETURNING *;

-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (user_id, family_id, token_hash, expires_at)
VALUES (@user_id, @family_id, @token_hash, @expires_at);

-- name: GetRefreshTokenForUpdate :one
SELECT * FROM refresh_tokens WHERE token_hash = @token_hash FOR UPDATE;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked_at = now() WHERE id = @id AND revoked_at IS NULL;

-- name: RevokeRefreshTokenFamily :exec
UPDATE refresh_tokens SET revoked_at = now() WHERE family_id = @family_id AND revoked_at IS NULL;

-- name: DeleteExpiredRefreshTokens :exec
DELETE FROM refresh_tokens WHERE expires_at < now();
