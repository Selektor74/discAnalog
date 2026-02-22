package users

import (
	"SelektorDisc/internal/domain/models"

	"github.com/google/uuid"
)

type UserRow struct {
	Id           uuid.UUID `db:"id"`
	Username     string    `db:"username"`
	PasswordHash string    `db:"password_hash"`
}

func (row *UserRow) Values() []any {
	return []any{
		row.Id, row.Username, row.PasswordHash,
	}
}

func ToModelUser(row *UserRow) *models.User {
	if row == nil {
		return nil
	}

	return &models.User{
		Id:           row.Id,
		Username:     row.Username,
		PasswordHash: row.PasswordHash,
	}
}

func FromModel(model *models.User) UserRow {
	if model == nil {
		return UserRow{}
	}
	return UserRow{
		Id:           model.Id,
		Username:     model.Username,
		PasswordHash: model.PasswordHash,
	}
}
