package model

import "github.com/google/uuid"

type User struct {
	Id           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
}

type UserStore interface {
	GetUser(id string) (User, error)
	GetUserByName(name string) (User, error)
}
