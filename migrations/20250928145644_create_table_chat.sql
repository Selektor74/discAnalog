-- +goose Up
-- +goose StatementBegin
SET search_path TO voicechat, public;

CREATE TABLE IF NOT EXISTS voicechat.chat_messages (
    message_id UUID PRIMARY KEY,
    room_id UUID REFERENCES voicechat.rooms(id) ON DELETE CASCADE,
    username TEXT,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS voicechat.chat_messages;
-- +goose StatementEnd
