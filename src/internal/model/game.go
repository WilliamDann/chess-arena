package model

import "github.com/google/uuid"

type Clock struct {
	ClockInitial   int `json:"clock_init_ms"`
	ClockIncrement int `json:"clock_inc_ms"`
	ClockWhite     int `json:"clock_white_ms"`
	ClockBlack     int `json:"clock_black_ms"`
	ClockLastHit   int `json:"clock_last_hit"`
}

type Game struct {
	Id          uuid.UUID `json:"id"`
	WhitePlayer uuid.UUID `json:"white_player"`
	BlackPlayer uuid.UUID `json:"black_player"`

	Clock
}
