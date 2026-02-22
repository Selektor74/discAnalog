package chat

import (
	"SelektorDisc/internal/domain/models"
	"SelektorDisc/internal/repository"
	"context"
	"strings"

	"github.com/Masterminds/squirrel"
)

type ChatRepository struct {
	*repository.BaseRepository
}

func NewChatRepository(r *repository.BaseRepository) *ChatRepository {
	return &ChatRepository{r}
}

func (repo *ChatRepository) CreateMessage(ctx context.Context, model *models.Chat) (*models.Chat, error) {
	row := FromModel(model)
	query := repo.StatementBuilder.
		Insert(chatTable).
		Columns(chatTableColumns...).
		Values(row.Values()...).
		Suffix("RETURNING " + strings.Join(chatTableColumns, ","))

	var out ChatRow

	if err := repo.Pool.Getx(ctx, &out, query); err != nil {
		return nil, err
	}
	return ToModelChat(&out), nil
}

func (repo *ChatRepository) GetChatMessages(ctx context.Context, roomId string, limit, offset uint64) ([]*models.Chat, error) {
	query := repo.StatementBuilder.Select(chatTableColumns...).From(chatTable).
		Where(squirrel.Eq{chatTableRoomId: roomId}).
		OrderBy("created_at ASC").
		Limit(limit).
		Offset(offset)

	var rows []*ChatRow

	if err := repo.Pool.Selectx(ctx, &rows, query); err != nil {
		return nil, err
	}

	messages := make([]*models.Chat, 0, len(rows))
	for _, row := range rows {
		messages = append(messages, ToModelChat(row))
	}

	return messages, nil
}
