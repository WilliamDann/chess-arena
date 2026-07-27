package services

import (
	"encoding/json"
	"net/http"

	"github.com/williamdann/chess-arena/internal/model"
	"github.com/williamdann/chess-arena/internal/postgres"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	users    *postgres.UserStore
	profiles *postgres.ProfileStore
}

func NewUserService(users *postgres.UserStore, profiles *postgres.ProfileStore) *UserService {
	return &UserService{users: users, profiles: profiles}
}

func (s *UserService) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/profile", s.createUser)
}

func (s *UserService) createUser(w http.ResponseWriter, r *http.Request) {
	// parse request body
	var requestData model.CreateUser
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "json body must contain email, password, and display_name", http.StatusBadRequest)
		return
	}

	// validate params
	err = requestData.Validate()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(requestData.Password), 12)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	requestData.Password = string(hash)

	// create user
	data, err := s.users.CreateUser(r.Context(), requestData)
	if err != nil {
		http.Error(w, "failed to create", http.StatusBadRequest)
		return
	}

	// return data
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
