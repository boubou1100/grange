-- Schéma PostgreSQL cible (pas encore branché : le MVP utilise meta.Memory).

CREATE TABLE repositories (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE CHECK (name ~ '^[a-zA-Z0-9][a-zA-Z0-9._-]{0,62}$'),
    format      TEXT NOT NULL,
    type        TEXT NOT NULL CHECK (type IN ('hosted', 'proxy', 'group')),
    attributes  JSONB NOT NULL DEFAULT '{}',   -- URL distante pour proxy, membres pour group...
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE components (
    id            BIGSERIAL PRIMARY KEY,
    repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    namespace     TEXT NOT NULL DEFAULT '',     -- groupId Maven, scope npm...
    name          TEXT NOT NULL,
    version       TEXT NOT NULL DEFAULT '',
    attributes    JSONB NOT NULL DEFAULT '{}',  -- licence, dépendances, etc. (parseur du format)
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (repository_id, namespace, name, version)
);

CREATE TABLE blobs (
    sha256     CHAR(64) PRIMARY KEY,
    size       BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE assets (
    id            BIGSERIAL PRIMARY KEY,
    repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    component_id  BIGINT REFERENCES components(id) ON DELETE CASCADE,
    path          TEXT NOT NULL,
    blob_sha256   CHAR(64) NOT NULL REFERENCES blobs(sha256),
    content_type  TEXT NOT NULL DEFAULT '',
    uploaded_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_download TIMESTAMPTZ,                  -- pour les politiques de nettoyage
    UNIQUE (repository_id, path)
);
CREATE INDEX assets_blob_idx ON assets(blob_sha256);  -- garbage collection des blobs orphelins

CREATE TABLE users (
    id       BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT,                         -- NULL si compte OIDC/LDAP
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE api_tokens (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,            -- on ne stocke jamais le jeton en clair
    name       TEXT NOT NULL,
    expires_at TIMESTAMPTZ
);

-- RBAC par dépôt : role = read | write | admin
CREATE TABLE permissions (
    user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    repository_id BIGINT REFERENCES repositories(id) ON DELETE CASCADE, -- NULL = tous les dépôts
    role          TEXT NOT NULL CHECK (role IN ('read', 'write', 'admin'))
);
