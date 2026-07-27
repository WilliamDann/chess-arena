package model

import (
	"errors"

	"github.com/google/uuid"
)

type User struct {
	Id           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-" db:"pw_hash"`
}

type CreateUser struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

// TODO real email validation
func (cu CreateUser) Validate() error {
	if len(cu.Email) < 1 {
		return errors.New("an email is required")
	}
	if len(cu.Password) < 5 {
		return errors.New("password must be >= 5 chars")
	}
	if len(cu.DisplayName) < 1 {
		return errors.New("a display_name must be defined")
	}

	return nil
}
