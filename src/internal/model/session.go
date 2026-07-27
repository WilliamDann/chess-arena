package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	Id        uuid.UUID `json:"id"`
	UserId    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type CreateSession struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (cs *CreateSession) Validate() error {
	if cs.Email == "" {
		return errors.New("email is missing")
	}
	if cs.Password == "" {
		return errors.New("password is missing")
	}
	return nil
}
