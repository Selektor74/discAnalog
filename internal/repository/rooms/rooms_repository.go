package rooms

import (
	"SelektorDisc/internal/domain/models"
	"SelektorDisc/internal/repository"
	"context"
	"errors"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

type RoomsRepository struct {
	*repository.BaseRepository
}

func NewRoomsRepository(r *repository.BaseRepository) *RoomsRepository {
	return &RoomsRepository{r}
}

func (repo *RoomsRepository) GetRoom(ctx context.Context, roomUuid string) (*models.Room, error) {

	query := repo.StatementBuilder.Select(roomsTableColumns...).From(roomsTable).Where(squirrel.Eq{roomsTableColumnUUID: roomUuid})

	var roomRow RoomRow

	if err := repo.Pool.Getx(ctx, &roomRow, query); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("Room not found")
		}
		return nil, err
	}
	return ToModelRoom(&roomRow), nil
}

func (repo *RoomsRepository) GetAllRooms(ctx context.Context) ([]*models.Room, error) {
	query := repo.StatementBuilder.Select(roomsTableColumns...).From(roomsTable)

	var roomRows []*RoomRow

	if err := repo.Pool.Selectx(ctx, &roomRows, query); err != nil {
		return nil, err
	}

	rooms := make([]*models.Room, 0, len(roomRows))
	for _, row := range roomRows {
		rooms = append(rooms, ToModelRoom(row))
	}
	return rooms, nil
}

func (repo *RoomsRepository) CreateRoom(ctx context.Context, model *models.Room) (*models.Room, error) {
	row := FromModel(model)

	query := repo.StatementBuilder.
		Insert(roomsTable).
		Columns(roomsTableColumns...).
		Values(row.Values()...).
		Suffix("RETURNING " + strings.Join(roomsTableColumns, ","))

	var out RoomRow
	if err := repo.Pool.Getx(ctx, &out, query); err != nil {
		return nil, err
	}
	return ToModelRoom(&out), nil
}
