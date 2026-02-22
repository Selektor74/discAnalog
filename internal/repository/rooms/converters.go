package rooms

import (
	"log"

	models "SelektorDisc/internal/domain/models" //todo remake to github

	"github.com/google/uuid"
)

type RoomRow struct {
	Id   string `db:"id"`
	Name string `db:"name"`
}

func (row *RoomRow) Values() []any {
	return []any{
		row.Id, row.Name,
	}
}

func ToModelRoom(row *RoomRow) *models.Room {
	if row == nil {
		return nil
	}

	parsedUUID, err := uuid.Parse(row.Id)
	if err != nil {
		log.Fatalf("Room UUID parsing is failed:%v", err)
	}
	return &models.Room{
		UUID: parsedUUID,
		Name: row.Name,
	}
}
func FromModel(model *models.Room) RoomRow {
	if model == nil {
		return RoomRow{}
	}
	return RoomRow{
		Id:   model.UUID.String(),
		Name: model.Name,
	}
}
