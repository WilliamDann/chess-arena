package events

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/williamdann/chess-arena/internal/model"
)

const TopicGame = "game"

func GameTopic(id uuid.UUID) string {
	return TopicGame + ":" + id.String()
}

type GameEventType string

const (
	GameMove  GameEventType = "game.move"
	GameState GameEventType = "game.state"
)

type MoveEvent struct {
	Type   GameEventType     `json:"type"`
	GameID uuid.UUID         `json:"game_id"`
	Move   model.Move        `json:"move"`
	Clock  model.ClockUpdate `json:"clock"`
}

func (e *MoveEvent) Topic() string {
	return GameTopic(e.GameID)
}
func (e *MoveEvent) Payload() ([]byte, error) {
	return json.Marshal(e)
}

func NewMoveEvent(move model.Move, clock model.ClockUpdate) *MoveEvent {
	return &MoveEvent{Type: GameMove, GameID: move.GameID, Move: move, Clock: clock}
}

// send to client directly - no pubsub
type GameStateEvent struct {
	Type  GameEventType `json:"type"`
	Game  model.Game    `json:"game"`
	Moves []model.Move  `json:"moves"`
}

func NewGameStateEvent(game model.Game, moves []model.Move) *GameStateEvent {
	return &GameStateEvent{Type: GameState, Game: game, Moves: moves}
}
