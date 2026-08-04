package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/williamdann/chess-arena/internal/model"
)

// maximum outstanding challenges a single user may have at once
const maxChallengesPerUser = 1

var ErrChallengeLimit = errors.New("challenge limit reached")
var ErrUnknownUser = errors.New("unknown user")

type ChallengeStore struct {
	db DB
}

func NewChallengeStore(db DB) *ChallengeStore {
	return &ChallengeStore{db: db}
}

// WithTx returns a copy of the store that runs its queries inside tx.
func (s *ChallengeStore) WithTx(tx pgx.Tx) *ChallengeStore {
	return &ChallengeStore{db: tx}
}

func (s *ChallengeStore) GetById(ctx context.Context, id uuid.UUID) (model.Challenge, error) {
	rows, err := s.db.Query(ctx, "select * from challenges where id = $1", id)
	if err != nil {
		return model.Challenge{}, err
	}

	data, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Challenge])
	if err != nil {
		return model.Challenge{}, err
	}

	return data, nil
}

func (s *ChallengeStore) GetOpen(ctx context.Context) ([]model.Challenge, error) {
	rows, err := s.db.Query(ctx, "select * from challenges where to_player is null")
	if err != nil {
		return []model.Challenge{}, err
	}

	data, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Challenge])
	if err != nil {
		return []model.Challenge{}, err
	}

	return data, nil
}

func (s *ChallengeStore) GetFromUser(ctx context.Context, userId uuid.UUID) ([]model.Challenge, error) {
	rows, err := s.db.Query(ctx, "select * from challenges where from_player = $1", userId)
	if err != nil {
		return []model.Challenge{}, err
	}

	data, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Challenge])
	if err != nil {
		return []model.Challenge{}, err
	}

	return data, nil
}

func (s *ChallengeStore) GetToUser(ctx context.Context, userId uuid.UUID) ([]model.Challenge, error) {
	rows, err := s.db.Query(ctx, "select * from challenges where to_player = $1", userId)
	if err != nil {
		return []model.Challenge{}, err
	}

	data, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Challenge])
	if err != nil {
		return []model.Challenge{}, err
	}

	return data, nil
}

func (s *ChallengeStore) Create(ctx context.Context, request model.CreateChallenge) (model.Challenge, error) {
	rows, err := s.db.Query(
		ctx,
		`insert into challenges (from_player, to_player, clock_initial_ms, clock_increment_ms)
		 select $1::uuid, $2::uuid, $3::int, $4::int
		 where (select count(*) from challenges where from_player = $1) < $5
		 returning *`,
		request.FromPlayer,
		request.ToPlayer,
		request.ClockInitial,
		request.ClockIncrement,
		maxChallengesPerUser,
	)

	if err != nil {
		return model.Challenge{}, err
	}

	challenge, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Challenge])
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Challenge{}, ErrChallengeLimit
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return model.Challenge{}, ErrUnknownUser
	}
	if err != nil {
		return model.Challenge{}, err
	}

	return challenge, nil
}

func (s *ChallengeStore) Delete(ctx context.Context, id uuid.UUID) (model.Challenge, error) {
	rows, err := s.db.Query(ctx, "delete from challenges where id = $1 returning *", id)
	if err != nil {
		return model.Challenge{}, err
	}

	challenge, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Challenge])
	if err != nil {
		return model.Challenge{}, err
	}

	return challenge, nil
}

func (s *ChallengeStore) DeleteForUser(ctx context.Context, itemId, userId uuid.UUID) (model.Challenge, error) {
	rows, err := s.db.Query(ctx, "delete from challenges where id = $2 and (from_player = $1 or to_player = $1) returning *", userId, itemId)
	if err != nil {
		return model.Challenge{}, err
	}

	challenge, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Challenge])
	if err != nil {
		return model.Challenge{}, err
	}

	return challenge, nil
}

func (s *ChallengeStore) DeleteOutgoing(ctx context.Context, userId uuid.UUID) ([]model.Challenge, error) {
	rows, err := s.db.Query(ctx, "delete from challenges where from_player = $1 returning *", userId)
	if err != nil {
		return []model.Challenge{}, err
	}

	data, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Challenge])
	if err != nil {
		return []model.Challenge{}, err
	}

	return data, err
}

func (s *ChallengeStore) DeleteIncoming(ctx context.Context, userId uuid.UUID) ([]model.Challenge, error) {
	rows, err := s.db.Query(ctx, "delete from challenges where to_player = $1 returning *", userId)
	if err != nil {
		return []model.Challenge{}, err
	}

	data, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Challenge])
	if err != nil {
		return []model.Challenge{}, err
	}

	return data, err
}

func (s *ChallengeStore) ClaimChallenge(ctx context.Context, itemId, userId uuid.UUID) (model.Challenge, error) {
	rows, err := s.db.Query(
		ctx,
		"delete from challenges where id = $1 and from_player <> $2 and (to_player is null or to_player = $2) returning *",
		itemId,
		userId,
	)

	if err != nil {
		return model.Challenge{}, err
	}

	challenge, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Challenge])
	if err != nil {
		return model.Challenge{}, err
	}

	return challenge, nil
}
