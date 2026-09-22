-- Accounts for admin panel users. The mobile app has no account: it only uses
-- the public route GET /drugs/ean/{ean}.
--   editor   -> creates and edits drugs, packages and leaflets
--   reviewer -> everything an editor can do + marks leaflets as reviewed (pharmacist)
--   admin    -> everything a reviewer can do + manages users
CREATE TABLE users
(
    id            BIGSERIAL PRIMARY KEY,
    email         VARCHAR(255) UNIQUE       NOT NULL, -- always lowercase
    name          TEXT                      NOT NULL,
    password_hash TEXT                      NOT NULL, -- bcrypt
    role          VARCHAR(20)               NOT NULL
        CONSTRAINT chk_users_role CHECK (role IN ('editor', 'reviewer', 'admin')),
    active        BOOLEAN     DEFAULT TRUE  NOT NULL, -- deactivate instead of delete to keep the review history
    created_at    TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at    TIMESTAMPTZ
);

-- Login sessions. Only the token's SHA-256 is stored: someone reading the
-- database can't use the sessions.
CREATE TABLE sessions
(
    token_hash BYTEA PRIMARY KEY,
    user_id    BIGINT                    NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    expires_at TIMESTAMPTZ               NOT NULL,

    CONSTRAINT fk_user_id
        FOREIGN KEY (user_id)
            REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_sessions_user_id ON sessions (user_id);

-- The review now points to the user who reviewed it. reviewed_by (text)
-- keeps their name as it was at review time.
ALTER TABLE summaries
    ADD COLUMN reviewed_by_user_id BIGINT
        CONSTRAINT fk_reviewed_by_user_id REFERENCES users (id) ON DELETE SET NULL;
