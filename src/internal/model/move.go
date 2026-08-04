package model

import (
	"time"

	"github.com/google/uuid"
)

type Move struct {
	GameID    uuid.UUID `json:"game_id"`
	Ply       int       `json:"ply"`
	UCI       string    `json:"uci"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateMove struct {
	GameID uuid.UUID `json:"game_id"`
	Ply    int       `json:"ply"`
	UCI    string    `json:"uci"`
}
