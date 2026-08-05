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
	// deleted because it was accepted; the game it produced rides along so
	// the challenger can join without polling
	ChallangeAccepted LobbyEventType = "lobby.accept"
)

type LobbyEvent struct {
	Type      LobbyEventType   `json:"type"`
	Id        uuid.UUID        `json:"id"`
	Challenge *model.Challenge `json:"challenge,omitempty"`
	Game      *model.Game      `json:"game,omitempty"`
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

func LobbyAcceptEvent(challenge model.Challenge, game model.Game) *LobbyEvent {
	return &LobbyEvent{Type: ChallangeAccepted, Id: challenge.Id, Game: &game}
}
