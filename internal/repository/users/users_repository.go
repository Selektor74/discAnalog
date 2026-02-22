package users

import (
	"SelektorDisc/internal/domain/models"
	"SelektorDisc/internal/repository"
	"context"
	"errors"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UserRepository struct {
	*repository.BaseRepository
}

func NewUsersRepository(r *repository.BaseRepository) *UserRepository {
	return &UserRepository{r}
}

func (repo *UserRepository) CreateUser(ctx context.Context, model *models.User) (*models.User, error) {
	row := FromModel(model)
	query := repo.StatementBuilder.
		Insert(usersTable).
		Columns(usersTableColumns...).
		Values(row.Values()...).
		Suffix("RETURNING " + strings.Join(usersTableColumns, ","))

	var out UserRow

	if err := repo.Pool.Getx(ctx, &out, query); err != nil {
		return nil, err
	}
	return ToModelUser(&out), nil
}

func (repo *UserRepository) GetUser(ctx context.Context, userUuid uuid.UUID) (*models.User, error) {
	query := repo.StatementBuilder.Select(usersTableColumns...).From(usersTable).Where(squirrel.Eq{usersTableColumnId: userUuid})

	var userRow UserRow

	if err := repo.Pool.Getx(ctx, &userRow, query); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("User not found")
		}
		return nil, err
	}
	return ToModelUser(&userRow), nil
}

func (repo *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	query := repo.StatementBuilder.
		Select(usersTableColumns...).
		From(usersTable).
		Where(squirrel.Eq{usersTableColumnUsername: username})

	var userRow UserRow
	if err := repo.Pool.Getx(ctx, &userRow, query); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("User not found")
		}
		return nil, err
	}
	return ToModelUser(&userRow), nil
}
