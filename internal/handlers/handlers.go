package handlers

import (
	"SelektorDisc/internal/repository/chat"
	"SelektorDisc/internal/repository/rooms"
	"SelektorDisc/internal/repository/users"
)

type Handlers struct {
	usersRepo *users.UserRepository
	roomsRepo *rooms.RoomsRepository
	chatRepo  *chat.ChatRepository
}

func New(u *users.UserRepository, r *rooms.RoomsRepository, c *chat.ChatRepository) *Handlers {
	return &Handlers{
		usersRepo: u,
		roomsRepo: r,
		chatRepo:  c,
	}
}
