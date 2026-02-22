package models

import "github.com/google/uuid"

type Room struct {
	UUID uuid.UUID
	Name string
}
