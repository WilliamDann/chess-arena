package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/williamdann/chess-arena/internal/model"
)

type MoveStore struct {
	db DB
}

func NewMoveStore(db DB) *MoveStore {
	return &MoveStore{db: db}
}

func (s *MoveStore) Create(ctx context.Context, request model.CreateMove) (model.Move, error) {
	rows, err := s.db.Query(
		ctx,
		"insert into moves (game_id, ply, uci, fen, white_ms, black_ms, created_at) values ($1, $2, $3, $4, $5, $6, $7) returning *",
		request.GameID,
		request.Ply,
		request.UCI,
		request.Fen,
		request.WhiteMS,
		request.BlackMS,
		request.CreatedAt,
	)

	if err != nil {
		return model.Move{}, err
	}

	move, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Move])
	if err != nil {
		return model.Move{}, err
	}

	return move, nil
}

func (s *MoveStore) GetLastMove(ctx context.Context, gameId uuid.UUID) (model.Move, error) {
	rows, err := s.db.Query(ctx, "select * from moves where game_id = $1 order by ply desc limit 1", gameId)
	if err != nil {
		return model.Move{}, err
	}

	move, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Move])
	if err != nil {
		return model.Move{}, err
	}

	return move, nil
}
func (s *MoveStore) GetForGame(ctx context.Context, gameId uuid.UUID) ([]model.Move, error) {
	rows, err := s.db.Query(ctx, "select * from moves where game_id = $1 order by ply", gameId)
	if err != nil {
		return []model.Move{}, err
	}

	moves, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Move])
	if err != nil {
		return []model.Move{}, err
	}

	return moves, nil
}

func (s *MoveStore) GetForGameAt(ctx context.Context, gameId uuid.UUID, ply int) (model.Move, error) {
	rows, err := s.db.Query(ctx, "select * from moves where game_id = $1 and ply = $2", gameId, ply)
	if err != nil {
		return model.Move{}, err
	}

	move, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Move])
	if err != nil {
		return model.Move{}, err
	}

	return move, nil
}
