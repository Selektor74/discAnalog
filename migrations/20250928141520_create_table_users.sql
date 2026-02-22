-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS voicechat;
SET search_path TO voicechat, public;

CREATE TABLE IF NOT EXISTS voicechat.users (
    id UUID PRIMARY KEY,
    username VARCHAR(100) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS voicechat.users;
-- +goose StatementEnd
