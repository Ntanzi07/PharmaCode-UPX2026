-- name: CreateSession :exec
INSERT INTO sessions (token_hash, user_id, expires_at)
VALUES ($1, $2, $3);

-- name: GetSessionUser :one
-- Só devolve a sessão se ela não expirou e o usuário continua ativo.
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
-- Usado na troca da própria senha: derruba as outras sessões e mantém a atual.
DELETE
FROM sessions
WHERE user_id = @user_id
  AND token_hash <> @keep_token_hash;
