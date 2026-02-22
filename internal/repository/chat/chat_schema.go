package chat

const chatTable = "chat_messages"

const (
	chatTableColumnId        = "message_id"
	chatTableRoomId          = "room_id"
	chatTableColumnUsername  = "username"
	chatTableColumnContent   = "content"
	chatTableColumnCreatedAt = "created_at"
)

var chatTableColumns = []string{
	chatTableColumnId,
	chatTableRoomId,
	chatTableColumnUsername,
	chatTableColumnContent,
	chatTableColumnCreatedAt,
}
