package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/williamdann/chess-arena/internal/model"
)

type ProfileStore struct {
	db *pgxpool.Pool
}

func NewProfileStore(db *pgxpool.Pool) *ProfileStore {
	return &ProfileStore{db: db}
}

func (s *ProfileStore) GetProfile(ctx context.Context, userId uuid.UUID) (model.Profile, error) {
	rows, err := s.db.Query(ctx, "select * from profiles where id = $1", userId)
	if err != nil {
		return model.Profile{}, err
	}

	data, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Profile])
	if err != nil {
		return model.Profile{}, err
	}

	return data, nil
}
