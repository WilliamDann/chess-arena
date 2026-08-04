package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Challenge struct {
	Id             uuid.UUID  `json:"id"`
	FromPlayer     uuid.UUID  `json:"from_player"`
	ToPlayer       *uuid.UUID `json:"to_player"`
	ClockInitial   int        `json:"clock_initial_ms" db:"clock_initial_ms"`
	ClockIncrement int        `json:"clock_increment_ms" db:"clock_increment_ms"`
	CreatedAt      time.Time  `json:"created_at"`
}

type CreateChallenge struct {
	FromPlayer     uuid.UUID  `json:"from_player"`
	ToPlayer       *uuid.UUID `json:"to_player"`
	ClockInitial   int        `json:"clock_initial_ms" db:"clock_initial_ms"`
	ClockIncrement int        `json:"clock_increment_ms" db:"clock_increment_ms"`
}

func (c *CreateChallenge) Validate() error {
	if c.ToPlayer != nil && c.FromPlayer == *c.ToPlayer {
		return errors.New("cannot play yourself")
	}

	if c.ClockInitial < int(time.Minute.Milliseconds()) || c.ClockInitial > int(time.Hour.Milliseconds())*5 {
		return errors.New("invalid initial time, must be between 1m and 5hr")
	}
	if c.ClockIncrement < 0 || c.ClockIncrement > int(time.Minute.Milliseconds())*15 {
		return errors.New("invalid increment, must be between 0 and 15m")
	}

	return nil
}

func (c *Challenge) Validate() error {
	if c.ToPlayer == nil {
		return errors.New("no ToPlayer exists")
	}
	if c.FromPlayer == *c.ToPlayer {
		return errors.New("cannot play yourself")
	}

	if c.ClockInitial < int(time.Minute.Milliseconds()) || c.ClockInitial > int(time.Hour.Milliseconds())*5 {
		return errors.New("invalid initial time, must be between 1m and 5hr")
	}
	if c.ClockIncrement < 0 || c.ClockIncrement > int(time.Minute.Milliseconds())*15 {
		return errors.New("invalid increment, must be between 0 and 15m")
	}

	return nil
}
