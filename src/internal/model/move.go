package model

import (
	"time"

	"github.com/google/uuid"
)

type MoveInfo struct {
	GameID    uuid.UUID `json:"game_id"`
	Ply       int       `json:"ply"`
	UCI       string    `json:"uci"`
	CreatedAt time.Time `json:"created_at"`
}

type SyncInfo struct {
	Fen     string `json:"fen"`
	WhiteMS int    `json:"white_ms"`
	BlackMS int    `json:"black_ms"`
}

type Move struct {
	MoveInfo
	SyncInfo
}

type SendMove struct {
	UCI string `json:"uci"`
}

type CreateMove struct {
	GameID    uuid.UUID `json:"game_id"`
	Ply       int       `json:"ply"`
	UCI       string    `json:"uci"`
	Fen       string    `json:"fen"`
	WhiteMS   int       `json:"white_ms"`
	BlackMS   int       `json:"black_ms"`
	CreatedAt time.Time `json:"created_at"`
}
