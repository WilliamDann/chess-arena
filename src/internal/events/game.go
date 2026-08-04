package events

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/williamdann/chess-arena/internal/model"
)

const TopicGame = "game"

// GameTopic is the per-game topic that game events are published on.
func GameTopic(id uuid.UUID) string {
	return TopicGame + ":" + id.String()
}

type GameEventType string

const (
	GameMove GameEventType = "game.move"
	GameEnd  GameEventType = "game.end"
)

type MoveEvent struct {
	Type GameEventType `json:"type"`
	Move model.Move    `json:"move"`
}

func (e *MoveEvent) Topic() string {
	return GameTopic(e.Move.GameID)
}
func (e *MoveEvent) Payload() ([]byte, error) {
	return json.Marshal(e)
}

func NewMoveEvent(move model.Move) *MoveEvent {
	return &MoveEvent{Type: GameMove, Move: move}
}

type EndEvent struct {
	Type   GameEventType `json:"type"`
	GameID uuid.UUID     `json:"game_id"`
	Result string        `json:"result"`
	Reason string        `json:"reason"`
}

func (e *EndEvent) Topic() string {
	return GameTopic(e.GameID)
}
func (e *EndEvent) Payload() ([]byte, error) {
	return json.Marshal(e)
}

func NewEndEvent(gameId uuid.UUID, result, reason string) *EndEvent {
	return &EndEvent{Type: GameEnd, GameID: gameId, Result: result, Reason: reason}
}
