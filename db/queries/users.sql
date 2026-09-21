-- name: CreateUser :one
INSERT INTO users (email, name, password_hash, role)
VALUES ($1, $2, $3, $4) RETURNING id;

-- name: GetUserByEmail :one
SELECT id, email, name, password_hash, role, active
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, name, role, active, created_at, updated_at
FROM users
WHERE id = $1;

-- name: ListUsers :many
SELECT id, email, name, role, active, created_at, updated_at
FROM users
ORDER BY name;

-- name: UpdateUser :execrows
UPDATE users
SET email      = $2,
    name       = $3,
    role       = $4,
    active     = $5,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateUserPassword :execrows
UPDATE users
SET password_hash = $2,
    updated_at    = NOW()
WHERE id = $1;

-- name: CountUsers :one
SELECT COUNT(*)
FROM users;

-- name: CountActiveAdmins :one
SELECT COUNT(*)
FROM users
WHERE role = 'admin'
  AND active;

-- name: GetUserPasswordHash :one
SELECT password_hash
FROM users
WHERE id = $1
  AND active;
