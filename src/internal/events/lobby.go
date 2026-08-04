package events

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/williamdann/chess-arena/internal/model"
)

const TopicLobby = "lobby"

type LobbyEventType string

const (
	ChallangeCreated LobbyEventType = "lobby.create"
	ChallangeDeleted LobbyEventType = "lobby.delete"
)

type LobbyEvent struct {
	Type      LobbyEventType   `json:"type"`
	Id        uuid.UUID        `json:"id"`
	Challenge *model.Challenge `json:"challenge,omitempty"`
}

func (e *LobbyEvent) Topic() string {
	return TopicLobby
}
func (e *LobbyEvent) Payload() ([]byte, error) {
	return json.Marshal(e)
}

func LobbyCreateEvent(data model.Challenge) *LobbyEvent {
	return &LobbyEvent{Type: ChallangeCreated, Id: data.Id, Challenge: &data}
}

func LobbyDeleteEvent(data model.Challenge) *LobbyEvent {
	return &LobbyEvent{Type: ChallangeDeleted, Id: data.Id}
}
