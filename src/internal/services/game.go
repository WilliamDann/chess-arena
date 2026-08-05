package services

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/williamdann/chess-arena/internal/model"
	"github.com/williamdann/chess-arena/internal/postgres"
)

type GameService struct {
	games *postgres.GameStore
	auth  func(http.Handler) http.Handler
}

func NewGameService(games *postgres.GameStore, auth func(http.Handler) http.Handler) *GameService {
	return &GameService{games: games, auth: auth}
}

func (s *GameService) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/game/{id}", s.getGame)

	mux.Handle("GET /api/games/live", s.auth(http.HandlerFunc(s.getActive)))
	mux.HandleFunc("GET /api/games/user/{id}", s.getGamesForUser)
}

func (s *GameService) getGame(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid game id", http.StatusBadRequest)
		return
	}

	game, err := s.games.GetById(r.Context(), id)
	if err != nil {
		http.Error(w, "unable to get game", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game)
}

func (s *GameService) getGamesForUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	params := r.URL.Query()
	limit, err := strconv.Atoi(params.Get("limit"))
	if err != nil {
		limit = 20
	}

	offset, err := strconv.Atoi(params.Get("offset"))
	if err != nil {
		offset = 0
	}

	data, err := s.games.GetForPlayer(r.Context(), id, limit, offset)
	if err != nil {
		http.Error(w, "get games failed", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *GameService) getActive(w http.ResponseWriter, r *http.Request) {
	session, ok := SessionFromContext(r.Context())
	if !ok {
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}

	games, err := s.games.GetActiveForPlayer(r.Context(), session.UserId)
	if err != nil {
		http.Error(w, "failed to get games", http.StatusInternalServerError)
		return
	}
	if games == nil {
		games = []model.Game{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(games)

}
