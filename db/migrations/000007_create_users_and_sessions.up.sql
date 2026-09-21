-- Contas de quem usa o painel admin. O app mobile não tem conta: ele só usa
-- a rota pública GET /drugs/ean/{ean}.
--   editor   -> cadastra e edita remédios, embalagens e bulas
--   reviewer -> tudo do editor + marca bula como revisada (farmacêutico)
--   admin    -> tudo do reviewer + gerencia usuários
CREATE TABLE users
(
    id            BIGSERIAL PRIMARY KEY,
    email         VARCHAR(255) UNIQUE       NOT NULL, -- sempre em minúsculas
    name          TEXT                      NOT NULL,
    password_hash TEXT                      NOT NULL, -- bcrypt
    role          VARCHAR(20)               NOT NULL
        CONSTRAINT chk_users_role CHECK (role IN ('editor', 'reviewer', 'admin')),
    active        BOOLEAN     DEFAULT TRUE  NOT NULL, -- desativar em vez de apagar preserva o histórico de revisões
    created_at    TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at    TIMESTAMPTZ
);

-- Sessões de login. Guardamos só o SHA-256 do token: quem ler o banco não
-- consegue usar as sessões.
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

-- A revisão passa a apontar para o usuário que revisou. O reviewed_by (texto)
-- continua como o nome dele no momento da revisão.
ALTER TABLE summaries
    ADD COLUMN reviewed_by_user_id BIGINT
        CONSTRAINT fk_reviewed_by_user_id REFERENCES users (id) ON DELETE SET NULL;
