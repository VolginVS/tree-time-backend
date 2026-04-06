-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    login TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS block_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    start_at TIMESTAMPTZ NOT NULL,
    finish_at TIMESTAMPTZ,
    block_range BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_block_sessions_user_start ON block_sessions(user_id, start_at DESC);

-- +goose Down
DROP TABLE IF EXISTS block_sessions;
DROP TABLE IF EXISTS users;
