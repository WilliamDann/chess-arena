package model

import (
	"time"

	"github.com/google/uuid"
)

type Clock struct {
	ClockInitial   int       `json:"clock_init_ms" db:"clock_initial_ms"`
	ClockIncrement int       `json:"clock_inc_ms" db:"clock_increment_ms"`
	ClockWhite     int       `json:"clock_white_ms" db:"clock_white_ms"`
	ClockBlack     int       `json:"clock_black_ms" db:"clock_black_ms"`
	ClockLastHit   time.Time `json:"clock_last_hit" db:"clock_last_hit"`
}

type Game struct {
	Id          uuid.UUID `json:"id"`
	WhitePlayer uuid.UUID `json:"white_player"`
	BlackPlayer uuid.UUID `json:"black_player"`

	Result   string     `json:"result"`
	ResultAt *time.Time `json:"result_at"`

	Clock

	CreatedAt time.Time `json:"created_at"`
}

type CreateGame struct {
	WhitePlayer    uuid.UUID `json:"white_player"`
	BlackPlayer    uuid.UUID `json:"black_player"`
	ClockInitial   int       `json:"clock_init_ms"`
	ClockIncrement int       `json:"clock_inc_ms"`
}
