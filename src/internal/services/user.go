package servies

import (
	"encoding/json"
	"net/http"

	"github.com/williamdann/chess-arena/internal/model"
	"github.com/williamdann/chess-arena/internal/postgres"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	users *postgres.UserStore
}

func NewUserService(users *postgres.UserStore) *UserService {
	return &UserService{users: users}
}

func (s *UserService) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/user", s.createUser)
	mux.HandleFunc("GET /api/user/{id}", s.getUser)
	mux.HandleFunc("GET /api/user", s.getUserEmail)
}

func (s *UserService) createUser(w http.ResponseWriter, r *http.Request) {
	// parse request body
	var requestData model.CreateUser
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "could not parse json request body", http.StatusBadRequest)
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

	// create user
	data, err := s.users.CreateUser(r.Context(), requestData.Email, string(hash))
	if err != nil {
		http.Error(w, "failed to create", http.StatusBadRequest)
		return
	}

	// return data
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *UserService) getUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	data, err := s.users.GetUserById(r.Context(), id)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *UserService) getUserEmail(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "email pram missing from request url", http.StatusBadRequest)
		return
	}

	data, err := s.users.GetUserByEmail(r.Context(), email)
	if err != nil {
		http.Error(w, "invalid email", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
