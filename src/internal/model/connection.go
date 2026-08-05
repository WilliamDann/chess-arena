package model

import (
	"time"

	"github.com/google/uuid"
)

type Connection struct {
	Id       uuid.UUID `json:"id"`
	UserId   uuid.UUID `json:"user_id"`
	ServerId uuid.UUID `json:"server_id"`
	LastSeen time.Time `json:"last_seen"`
}
