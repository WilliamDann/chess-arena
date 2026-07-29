package model

import (
	"time"

	"github.com/google/uuid"
)

type Challenge struct {
	Id             uuid.UUID  `json:"id"`
	FromPlayer     uuid.UUID  `json:"from_player"`
	ToPlayer       *uuid.UUID `json:"to_player"`
	ClockInitial   int        `json:"clock_init_ms" db:"clock_init_ms"`
	ClockIncrement int        `json:"clock_inc_ms" db:"clock_inc_ms"`
	CreatedAt      time.Time  `json:"created_at"`
}

type CreateChallenge struct {
	FromPlayer     uuid.UUID  `json:"from_player"`
	ToPlayer       *uuid.UUID `json:"to_player"`
	ClockInitial   int        `json:"clock_init_ms" db:"clock_init_ms"`
	ClockIncrement int        `json:"clock_inc_ms" db:"clock_inc_ms"`
}
