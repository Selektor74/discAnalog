package chat

import (
	"SelektorDisc/internal/domain/models"
	"log"
	"time"

	"github.com/google/uuid"
)

type ChatRow struct {
	Id       string    `db:"message_id"`
	RoomId   string    `db:"room_id"`
	Content  string    `db:"content"`
	Date     time.Time `db:"created_at"`
	Username string    `db:"username"`
}

func (row *ChatRow) Values() []any {
	return []any{
		row.Id, row.RoomId, row.Username, row.Content, row.Date,
	}
}

func ToModelChat(row *ChatRow) *models.Chat {
	if row == nil {
		return nil
	}

	parsedUUID, err := uuid.Parse(row.Id)

	if err != nil {
		log.Fatalf("Chat UUID parsing is failed:%v", err)
	}

	parseRoomdUUID, err := uuid.Parse(row.RoomId)
	if err != nil {
		log.Fatalf("Room UUID parsing is failed:%v", err)
	}
	return &models.Chat{
		Id:       parsedUUID,
		RoomId:   parseRoomdUUID,
		Content:  row.Content,
		Date:     row.Date,
		Username: row.Username,
	}
}
func FromModel(model *models.Chat) ChatRow {
	if model == nil {
		return ChatRow{}
	}
	return ChatRow{
		Id:       model.Id.String(),
		RoomId:   model.RoomId.String(),
		Content:  model.Content,
		Date:     model.Date,
		Username: model.Username,
	}
}
