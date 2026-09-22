-- name: CreateSession :exec
INSERT INTO sessions (token_hash, user_id, expires_at)
VALUES ($1, $2, $3);

-- name: GetSessionUser :one
-- Only returns the session if it hasn't expired and the user is still active.
SELECT u.id, u.email, u.name, u.role
FROM sessions AS s
         INNER JOIN users AS u ON u.id = s.user_id
WHERE s.token_hash = $1
  AND s.expires_at > NOW()
  AND u.active;

-- name: DeleteSession :exec
DELETE
FROM sessions
WHERE token_hash = $1;

-- name: DeleteUserSessions :exec
DELETE
FROM sessions
WHERE user_id = $1;

-- name: DeleteExpiredSessions :exec
DELETE
FROM sessions
WHERE expires_at <= NOW();

-- name: DeleteUserSessionsExcept :exec
-- Used when users change their own password: ends their other sessions and keeps the current one.
DELETE
FROM sessions
WHERE user_id = @user_id
  AND token_hash <> @keep_token_hash;
