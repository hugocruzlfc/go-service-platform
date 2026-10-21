CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    disabled BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT users_email_not_blank CHECK (char_length(btrim(email)) > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS users_email_lower_unique
    ON users (lower(email));

CREATE TABLE IF NOT EXISTS user_credentials (
    user_id TEXT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    password_hash TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS roles (
    name TEXT PRIMARY KEY,
    CONSTRAINT roles_name_not_blank CHECK (char_length(btrim(name)) > 0)
);

CREATE TABLE IF NOT EXISTS permissions (
    name TEXT PRIMARY KEY,
    CONSTRAINT permissions_name_not_blank CHECK (char_length(btrim(name)) > 0)
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_name TEXT NOT NULL REFERENCES roles (name) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_name)
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_name TEXT NOT NULL REFERENCES roles (name) ON DELETE CASCADE,
    permission_name TEXT NOT NULL REFERENCES permissions (name) ON DELETE CASCADE,
    PRIMARY KEY (role_name, permission_name)
);

CREATE TABLE IF NOT EXISTS auth_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_family TEXT NOT NULL,
    refresh_token_hash TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    CONSTRAINT auth_sessions_token_family_not_blank
        CHECK (char_length(btrim(token_family)) > 0),
    CONSTRAINT auth_sessions_expiry_after_creation
        CHECK (expires_at > created_at)
);

CREATE INDEX IF NOT EXISTS auth_sessions_user_id_idx
    ON auth_sessions (user_id);

CREATE INDEX IF NOT EXISTS auth_sessions_token_family_idx
    ON auth_sessions (token_family);

INSERT INTO roles (name)
VALUES ('user'), ('admin')
ON CONFLICT (name) DO NOTHING;

INSERT INTO permissions (name)
VALUES ('tasks:read'), ('tasks:write'), ('tasks:delete')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name)
VALUES
    ('user', 'tasks:read'),
    ('user', 'tasks:write'),
    ('user', 'tasks:delete'),
    ('admin', 'tasks:read'),
    ('admin', 'tasks:write'),
    ('admin', 'tasks:delete')
ON CONFLICT (role_name, permission_name) DO NOTHING;
