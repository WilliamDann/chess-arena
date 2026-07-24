package model

import (
	"github.com/google/uuid"
	"time"
)

type Session struct {
	Id        uuid.UUID `json:"id"`
	UserId    uuid.UUid `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
