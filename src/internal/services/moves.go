package services

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/williamdann/chess-arena/internal/postgres"
)

type MoveService struct {
	moves *postgres.MoveStore
}

func NewMoveService(moves *postgres.MoveStore) *MoveService {
	return &MoveService{moves: moves}
}

func (s *MoveService) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/game/{id}/moves", s.getMoves)
}

func (s *MoveService) getMoves(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid game id", http.StatusBadRequest)
		return
	}

	moves, err := s.moves.GetForGame(r.Context(), id)
	if err != nil {
		http.Error(w, "invalid game id", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(moves)
}
