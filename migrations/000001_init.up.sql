-- ============================================================
-- 000001_init.up.sql – Complete ChatGo schema
-- ============================================================

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";   -- for ILIKE index support

-- ============================================================
-- ROLES & PERMISSIONS
-- ============================================================
CREATE TABLE roles (
    id          UUID PRIMARY KEY,
    name        VARCHAR(64)  NOT NULL UNIQUE,
    description TEXT         NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE permissions (
    id       UUID PRIMARY KEY,
    name     VARCHAR(128) NOT NULL UNIQUE,
    resource VARCHAR(64)  NOT NULL,
    action   VARCHAR(64)  NOT NULL
);

CREATE TABLE role_permissions (
    role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- Seed default roles
INSERT INTO roles (id, name, description) VALUES
    ('00000000-0000-0000-0000-000000000001', 'member',  'Regular member'),
    ('00000000-0000-0000-0000-000000000002', 'admin',   'System administrator'),
    ('00000000-0000-0000-0000-000000000003', 'moderator','Content moderator');

-- ============================================================
-- USERS
-- ============================================================
CREATE TABLE users (
    id            UUID         PRIMARY KEY,
    username      VARCHAR(32)  NOT NULL UNIQUE,
    email         VARCHAR(254) NOT NULL UNIQUE,
    password_hash TEXT         NOT NULL,
    display_name  VARCHAR(64)  NOT NULL,
    avatar_url    TEXT         NOT NULL DEFAULT '',
    status        VARCHAR(16)  NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active','inactive','locked')),
    role_id       UUID         NOT NULL REFERENCES roles(id),
    mfa_enabled   BOOLEAN      NOT NULL DEFAULT FALSE,
    mfa_secret    TEXT         NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX idx_users_username    ON users(username) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_email       ON users(email)    WHERE deleted_at IS NULL;
CREATE INDEX idx_users_status      ON users(status)   WHERE deleted_at IS NULL;
CREATE INDEX idx_users_search      ON users USING gin(
    (username || ' ' || display_name) gin_trgm_ops
) WHERE deleted_at IS NULL;

-- ============================================================
-- SESSIONS
-- ============================================================
CREATE TABLE sessions (
    id                  UUID         PRIMARY KEY,
    user_id             UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash  TEXT         NOT NULL UNIQUE,
    ip_address          VARCHAR(45)  NOT NULL DEFAULT '',
    user_agent          TEXT         NOT NULL DEFAULT '',
    expires_at          TIMESTAMPTZ  NOT NULL,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    revoked_at          TIMESTAMPTZ
);

CREATE INDEX idx_sessions_user_id    ON sessions(user_id);
CREATE INDEX idx_sessions_refresh    ON sessions(refresh_token_hash);
CREATE INDEX idx_sessions_active     ON sessions(user_id, expires_at)
    WHERE revoked_at IS NULL;

-- ============================================================
-- CONVERSATIONS
-- ============================================================
CREATE TABLE conversations (
    id            UUID         PRIMARY KEY,
    type          VARCHAR(16)  NOT NULL CHECK (type IN ('direct','group','channel')),
    name          VARCHAR(128) NOT NULL DEFAULT '',
    description   TEXT         NOT NULL DEFAULT '',
    channel_type  VARCHAR(32)  CHECK (channel_type IN ('public','private','moderated','announcement')),
    is_invite_only BOOLEAN     NOT NULL DEFAULT FALSE,
    creator_id    UUID         NOT NULL REFERENCES users(id),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX idx_conversations_type       ON conversations(type) WHERE deleted_at IS NULL;
CREATE INDEX idx_conversations_creator    ON conversations(creator_id);
CREATE INDEX idx_conversations_name_trgm  ON conversations USING gin(name gin_trgm_ops)
    WHERE deleted_at IS NULL AND type = 'channel';

-- ============================================================
-- PARTICIPANTS
-- ============================================================
CREATE TABLE participants (
    id                    UUID        PRIMARY KEY,
    conversation_id       UUID        NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id               UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role                  VARCHAR(16) NOT NULL DEFAULT 'member'
                          CHECK (role IN ('member','admin','owner')),
    joined_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_read_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_muted              BOOLEAN     NOT NULL DEFAULT FALSE,
    notifications_enabled BOOLEAN     NOT NULL DEFAULT TRUE,
    UNIQUE (conversation_id, user_id)
);

CREATE INDEX idx_participants_user     ON participants(user_id);
CREATE INDEX idx_participants_conv     ON participants(conversation_id);

-- ============================================================
-- MESSAGES
-- ============================================================
CREATE TABLE messages (
    id              UUID        PRIMARY KEY,
    conversation_id UUID        NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id       UUID        NOT NULL REFERENCES users(id),
    content         TEXT        NOT NULL DEFAULT '',
    content_type    VARCHAR(16) NOT NULL DEFAULT 'text'
                    CHECK (content_type IN ('text','file','system')),
    parent_id       UUID        REFERENCES messages(id) ON DELETE SET NULL,
    is_edited       BOOLEAN     NOT NULL DEFAULT FALSE,
    is_deleted      BOOLEAN     NOT NULL DEFAULT FALSE,
    is_pinned       BOOLEAN     NOT NULL DEFAULT FALSE,
    metadata        JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_messages_conversation ON messages(conversation_id, created_at DESC)
    WHERE is_deleted = FALSE;
CREATE INDEX idx_messages_sender       ON messages(sender_id);
CREATE INDEX idx_messages_pinned       ON messages(conversation_id) WHERE is_pinned = TRUE;
CREATE INDEX idx_messages_fts          ON messages USING gin(
    to_tsvector('english', content)
) WHERE is_deleted = FALSE;

-- ============================================================
-- MESSAGE STATUS (delivery / read receipts)
-- ============================================================
CREATE TABLE message_status (
    message_id  UUID        NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status      VARCHAR(16) NOT NULL DEFAULT 'sent'
                CHECK (status IN ('sent','delivered','read')),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id)
);

CREATE INDEX idx_msg_status_user ON message_status(user_id);

-- ============================================================
-- REACTIONS
-- ============================================================
CREATE TABLE reactions (
    id         UUID        PRIMARY KEY,
    message_id UUID        NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    emoji      VARCHAR(16) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (message_id, user_id, emoji)
);

CREATE INDEX idx_reactions_message ON reactions(message_id);

-- ============================================================
-- FILES
-- ============================================================
CREATE TABLE files (
    id               UUID        PRIMARY KEY,
    name             VARCHAR(512) NOT NULL,
    original_name    VARCHAR(512) NOT NULL,
    mime_type        VARCHAR(128) NOT NULL,
    size             BIGINT      NOT NULL,
    storage_path     TEXT        NOT NULL,
    uploader_id      UUID        NOT NULL REFERENCES users(id),
    conversation_id  UUID        REFERENCES conversations(id) ON DELETE SET NULL,
    is_scanned       BOOLEAN     NOT NULL DEFAULT FALSE,
    scan_result      VARCHAR(16) NOT NULL DEFAULT 'skipped'
                     CHECK (scan_result IN ('clean','infected','pending','skipped')),
    access_level     VARCHAR(16) NOT NULL DEFAULT 'private'
                     CHECK (access_level IN ('public','private','restricted')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ
);

CREATE INDEX idx_files_uploader ON files(uploader_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_files_conv     ON files(conversation_id) WHERE deleted_at IS NULL;

-- ============================================================
-- NOTIFICATIONS
-- ============================================================
CREATE TABLE notifications (
    id             UUID        PRIMARY KEY,
    user_id        UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type           VARCHAR(16) NOT NULL CHECK (type IN ('message','mention','system','security')),
    title          VARCHAR(256) NOT NULL,
    body           TEXT        NOT NULL DEFAULT '',
    reference_id   VARCHAR(256) NOT NULL DEFAULT '',
    reference_type VARCHAR(64)  NOT NULL DEFAULT '',
    is_read        BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    read_at        TIMESTAMPTZ
);

CREATE INDEX idx_notif_user_unread ON notifications(user_id, created_at DESC)
    WHERE is_read = FALSE;

-- ============================================================
-- ACTIVITY LOGS
-- ============================================================
CREATE TABLE activity_logs (
    id            UUID        PRIMARY KEY,
    user_id       UUID        REFERENCES users(id) ON DELETE SET NULL,
    action        VARCHAR(64) NOT NULL,
    resource_type VARCHAR(64) NOT NULL DEFAULT '',
    resource_id   VARCHAR(256) NOT NULL DEFAULT '',
    details       JSONB,
    ip_address    VARCHAR(45) NOT NULL DEFAULT '',
    user_agent    TEXT        NOT NULL DEFAULT '',
    severity      VARCHAR(16) NOT NULL DEFAULT 'info'
                  CHECK (severity IN ('info','warning','error','critical')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_activity_user       ON activity_logs(user_id, created_at DESC);
CREATE INDEX idx_activity_action     ON activity_logs(action);
CREATE INDEX idx_activity_severity   ON activity_logs(severity);
CREATE INDEX idx_activity_created    ON activity_logs(created_at DESC);
CREATE INDEX idx_activity_resource   ON activity_logs(resource_type, resource_id);

-- ============================================================
-- SCHEMA MIGRATIONS TABLE (managed by golang-migrate)
-- ============================================================
-- Note: golang-migrate creates schema_migrations automatically.
-- The above is the full application schema.
