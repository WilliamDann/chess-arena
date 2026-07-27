package postgres

import (
	"context"

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

func (s *UserStore) GetUser(ctx context.Context, id string) (model.User, error) {
	rows, err := s.db.Query(ctx, "select * from users where id = $1", id)
	if err != nil {
		return model.User{}, err
	}

	data, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.User])
	if err != nil {
		return model.User{}, err
	}

	return data, err
}

func (s *UserStore) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	rows, err := s.db.Query(ctx, "select * from users where email=$1", email)
	if err != nil {
		return model.User{}, nil
	}

	data, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.User])
	if err != nil {
		return model.User{}, err
	}

	return data, err
}
