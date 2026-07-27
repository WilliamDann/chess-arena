package model

import "github.com/google/uuid"

type User struct {
	Id           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-" db:"pw_hash"`
}

type CreateUser struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
