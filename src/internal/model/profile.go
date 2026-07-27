package model

import "github.com/google/uuid"

type Profile struct {
	Id          uuid.UUID `json:"id"`
	DisplayName *string   `json:"display_name"`
}
