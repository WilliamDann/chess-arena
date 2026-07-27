package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/williamdann/chess-arena/internal/model"
)

type UserStore struct {
	db *pgxpool.Pool
}

func NewUserStore(db *pgxpool.Pool) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) CreateUser(ctx context.Context, data model.CreateUser) (model.User, error) {
	// start transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return model.User{}, err
	}
	defer tx.Rollback(ctx)

	// insert user
	rows, err := tx.Query(ctx, "insert into users (email, pw_hash) values($1, $2) returning *", data.Email, data.Password)
	if err != nil {
		return model.User{}, errors.New("item not found")
	}
	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.User])
	if err != nil {
		return model.User{}, err
	}

	// insert profile
	_, err = tx.Exec(ctx, "insert into profiles (id, display_name) values ($1, $2)", user.Id, data.DisplayName)
	if err != nil {
		return model.User{}, err
	}

	// commit transaction and return created
	return user, tx.Commit(ctx)
}

func (s *UserStore) GetUserById(ctx context.Context, id uuid.UUID) (model.User, error) {
	rows, err := s.db.Query(ctx, "select * from users where id = $1", id)
	if err != nil {
		return model.User{}, errors.New("item not found")
	}

	data, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.User])
	if err != nil {
		return model.User{}, err
	}

	return data, nil
}

func (s *UserStore) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	rows, err := s.db.Query(ctx, "select * from users where email = $1", email)
	if err != nil {
		return model.User{}, errors.New("item not found")
	}

	data, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.User])
	if err != nil {
		return model.User{}, err
	}

	return data, nil
}
