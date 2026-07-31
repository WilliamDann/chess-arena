package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/williamdann/chess-arena/internal/model"
)

type ProfileStore struct {
	db DB
}

func NewProfileStore(db DB) *ProfileStore {
	return &ProfileStore{db: db}
}

// WithTx returns a copy of the store that runs its queries inside tx.
func (s *ProfileStore) WithTx(tx pgx.Tx) *ProfileStore {
	return &ProfileStore{db: tx}
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
