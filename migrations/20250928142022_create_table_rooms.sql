-- +goose Up
-- +goose StatementBegin
SET search_path TO voicechat, public;

CREATE TABLE IF NOT EXISTS voicechat.rooms (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS voicechat.rooms_members (
    user_id UUID REFERENCES voicechat.users(id),
    room_id UUID REFERENCES voicechat.rooms(id),
    PRIMARY KEY (user_id, room_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS voicechat.rooms_members;
DROP TABLE IF EXISTS voicechat.rooms;
-- +goose StatementEnd
