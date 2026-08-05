package events

import (
	"encoding/json"

	"github.com/williamdann/chess-arena/internal/model"
)

const TopicPresence = "presence"

type PresenceEventType string

const (
	PresenceJoin  PresenceEventType = "presence.join"
	PresenceLeave PresenceEventType = "presence.leave"
)

type PresenceEvent struct {
	Type    PresenceEventType `json:"type"`
	Profile model.Profile     `json:"profile"`
}

func (e *PresenceEvent) Topic() string {
	return TopicPresence
}

func (e *PresenceEvent) Payload() ([]byte, error) {
	return json.Marshal(e)
}

func JoinPresenceEvent(profile model.Profile) *PresenceEvent {
	return &PresenceEvent{Type: PresenceJoin, Profile: profile}
}
func LeavePresenceEvent(profile model.Profile) *PresenceEvent {
	return &PresenceEvent{Type: PresenceLeave, Profile: profile}
}
