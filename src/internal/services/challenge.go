package services

import (
	"encoding/json"
	"errors"
	"log/slog"
	"math/rand/v2"
	"net/http"

	"github.com/google/uuid"
	"github.com/williamdann/chess-arena/internal/events"
	"github.com/williamdann/chess-arena/internal/model"
	"github.com/williamdann/chess-arena/internal/postgres"
	"github.com/williamdann/chess-arena/internal/pubsub"
)

type ChallengeService struct {
	challenges *postgres.ChallengeStore
	games      *postgres.GameStore
	pubsub     *pubsub.PubSub
	auth       func(http.Handler) http.Handler
}

func NewChallengeService(store *postgres.ChallengeStore, games *postgres.GameStore, pubsub *pubsub.PubSub, auth func(http.Handler) http.Handler) *ChallengeService {
	return &ChallengeService{challenges: store, games: games, pubsub: pubsub, auth: auth}
}

func (s *ChallengeService) Register(mux *http.ServeMux) {
	mux.Handle("GET /api/challenges/open", s.auth(http.HandlerFunc(s.getOpen)))

	mux.Handle("GET /api/challenges/out", s.auth(http.HandlerFunc(s.getOutgoing)))
	mux.Handle("DELETE /api/challenges/out", s.auth(http.HandlerFunc(s.deleteOutgoing)))

	mux.Handle("GET /api/challenges/in", s.auth(http.HandlerFunc(s.getIncoming)))
	mux.Handle("DELETE /api/challenges/in", s.auth(http.HandlerFunc(s.deleteIncoming)))

	mux.Handle("GET /api/challenge/{id}", s.auth(http.HandlerFunc(s.getItem)))
	mux.Handle("POST /api/challenge", s.auth(http.HandlerFunc(s.create)))
	mux.Handle("DELETE /api/challenge/{id}", s.auth(http.HandlerFunc(s.deleteItem)))

	mux.Handle("POST /api/challenge/accept/{id}", s.auth(http.HandlerFunc(s.accept)))
}

func (s *ChallengeService) getItem(w http.ResponseWriter, r *http.Request) {
	stringId := r.PathValue("id")
	id, err := uuid.Parse(stringId)
	if err != nil {
		http.Error(w, "invalid item id", http.StatusBadRequest)
		return
	}

	item, err := s.challenges.GetById(r.Context(), id)
	if err != nil {
		http.Error(w, "unable to find challenge", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func (s *ChallengeService) getOpen(w http.ResponseWriter, r *http.Request) {
	data, err := s.challenges.GetOpen(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *ChallengeService) getIncoming(w http.ResponseWriter, r *http.Request) {
	me, ok := SessionFromContext(r.Context())
	if !ok {
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}

	data, err := s.challenges.GetToUser(r.Context(), me.UserId)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *ChallengeService) getOutgoing(w http.ResponseWriter, r *http.Request) {
	me, ok := SessionFromContext(r.Context())
	if !ok {
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}

	data, err := s.challenges.GetFromUser(r.Context(), me.UserId)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *ChallengeService) create(w http.ResponseWriter, r *http.Request) {
	// get current user
	me, ok := SessionFromContext(r.Context())
	if !ok {
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}

	// parse challenge
	var request model.CreateChallenge
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "error parsing request body", http.StatusBadRequest)
		return
	}

	if request.FromPlayer != uuid.Nil && request.FromPlayer != me.UserId {
		http.Error(w, "you can only create challenges as yourself", http.StatusForbidden)
		return
	}

	// set FromPlayer to session user
	request.FromPlayer = me.UserId

	err = request.Validate()
	if err != nil {
		http.Error(w, "invalid challenge: "+err.Error(), http.StatusBadRequest)
		return
	}

	// create object
	data, err := s.challenges.Create(r.Context(), request)
	if errors.Is(err, postgres.ErrChallengeLimit) {
		http.Error(w, "too many open challenges", http.StatusConflict)
		return
	}
	if errors.Is(err, postgres.ErrUnknownUser) {
		http.Error(w, "challenged user does not exist", http.StatusBadRequest)
		return
	}
	if err != nil {
		slog.Error("challenge insert failed", "err", err)
		http.Error(w, "failed to create challenge", http.StatusInternalServerError)
		return
	}

	// return result
	s.pubsub.Pub(r.Context(), events.LobbyCreateEvent(data))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(data)
}

func (s *ChallengeService) deleteItem(w http.ResponseWriter, r *http.Request) {
	me, ok := SessionFromContext(r.Context())
	if !ok {
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}

	idString := r.PathValue("id")
	id, err := uuid.Parse(idString)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	data, err := s.challenges.DeleteForUser(r.Context(), id, me.UserId)
	if err != nil {
		http.Error(w, "failed to delete", http.StatusBadRequest)
		return
	}

	s.pubsub.Pub(r.Context(), events.LobbyDeleteEvent(data))
	w.WriteHeader(http.StatusOK)
}

func (s *ChallengeService) deleteIncoming(w http.ResponseWriter, r *http.Request) {
	me, ok := SessionFromContext(r.Context())
	if !ok {
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}

	data, err := s.challenges.DeleteIncoming(r.Context(), me.UserId)
	if err != nil {
		http.Error(w, "failed to delete", http.StatusInternalServerError)
		return
	}

	for _, item := range data {
		s.pubsub.Pub(r.Context(), events.LobbyDeleteEvent(item))
	}
	w.WriteHeader(http.StatusOK)
}

func (s *ChallengeService) deleteOutgoing(w http.ResponseWriter, r *http.Request) {
	me, ok := SessionFromContext(r.Context())
	if !ok {
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}

	data, err := s.challenges.DeleteOutgoing(r.Context(), me.UserId)
	if err != nil {
		http.Error(w, "failed to delete", http.StatusInternalServerError)
		return
	}

	for _, item := range data {
		s.pubsub.Pub(r.Context(), events.LobbyDeleteEvent(item))
	}
	w.WriteHeader(http.StatusOK)
}

func (s *ChallengeService) accept(w http.ResponseWriter, r *http.Request) {
	// parse request
	me, ok := SessionFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	stringId := r.PathValue("id")
	id, err := uuid.Parse(stringId)
	if err != nil {
		http.Error(w, "invalid challenge id", http.StatusBadRequest)
		return
	}

	// claim challange
	challenge, err := s.challenges.ClaimChallenge(r.Context(), id, me.UserId)
	if err != nil {
		http.Error(w, "unable to accept challenge", http.StatusNotFound)
		return
	}

	// coin flip for white/black
	whitePlayer := challenge.FromPlayer
	blackPlayer := me.UserId
	if rand.N(2) == 0 {
		blackPlayer, whitePlayer = whitePlayer, blackPlayer
	}

	// create the game
	request := model.CreateGame{
		WhitePlayer:    whitePlayer,
		BlackPlayer:    blackPlayer,
		ClockInitial:   challenge.ClockInitial,
		ClockIncrement: challenge.ClockIncrement,
	}
	game, err := s.games.Create(r.Context(), request)
	if err != nil {
		// the claim already removed the challenge, so still tell the lobby
		s.pubsub.Pub(r.Context(), events.LobbyDeleteEvent(challenge))
		slog.Error("failed to create game", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// published only after the game exists, so the challenger can join it
	// directly off the event
	s.pubsub.Pub(r.Context(), events.LobbyAcceptEvent(challenge, game))

	// send result
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game)
}
