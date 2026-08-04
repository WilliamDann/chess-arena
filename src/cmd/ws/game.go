package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/notnil/chess"
	"github.com/williamdann/chess-arena/internal/events"
	"github.com/williamdann/chess-arena/internal/model"
	"github.com/williamdann/chess-arena/internal/postgres"
	"github.com/williamdann/chess-arena/internal/pubsub"
	"github.com/williamdann/chess-arena/internal/services"
)

// clientMsg is the envelope for messages sent on the game socket. An empty
// type means "move" so bare {"uci": ...} messages keep working.
type clientMsg struct {
	Type string `json:"type"`
	model.SendMove
}

// gameConn is one player's (or spectator's) connection to a game.
type gameConn struct {
	games  *postgres.GameStore
	moves  *postgres.MoveStore
	ps     *pubsub.PubSub
	game   model.Game
	player uuid.UUID
	ended  atomic.Bool
}

func wsGame(games *postgres.GameStore, moves *postgres.MoveStore, ps *pubsub.PubSub) wsHandler {
	return func(conn *websocket.Conn, r *http.Request) {
		gameId, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			slog.Error("invalid game id", "err", err)
			return
		}

		session, ok := services.SessionFromContext(r.Context())
		if !ok {
			slog.Error("no session on game connection")
			return
		}

		// players and clock config are immutable, so one fetch serves the
		// whole connection; result changes arrive as game.end events
		game, err := games.GetById(r.Context(), gameId)
		if err != nil {
			slog.Error("unknown game", "game", gameId, "err", err)
			return
		}

		gc := &gameConn{games: games, moves: moves, ps: ps, game: game, player: session.UserId}
		gc.ended.Store(game.Result != "*")

		ch, drop := ps.Sub(events.GameTopic(gameId))
		defer drop()

		go func() {
			defer drop() // closes ch, which unblocks the write loop below
			for {
				_, data, err := conn.ReadMessage()
				if err != nil {
					return
				}
				if err := gc.handle(r.Context(), data); err != nil {
					slog.Error("rejected message", "game", gameId, "err", err)
				}
			}
		}()

		for msg := range ch {
			gc.observe(msg.Payload)
			if err := conn.WriteMessage(websocket.TextMessage, msg.Payload); err != nil {
				slog.Error("err sending game message", "err", err)
				return
			}
		}
	}
}

// observe watches outbound events so the connection learns when the game ends.
func (c *gameConn) observe(payload []byte) {
	var head struct {
		Type events.GameEventType `json:"type"`
	}
	if json.Unmarshal(payload, &head) == nil && head.Type == events.GameEnd {
		c.ended.Store(true)
	}
}

func (c *gameConn) handle(ctx context.Context, data []byte) error {
	var msg clientMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("parsing message: %w", err)
	}

	if c.ended.Load() {
		return errors.New("game is over")
	}

	switch msg.Type {
	case "", "move":
		return c.handleMove(ctx, msg.SendMove)
	case "flag":
		return c.handleFlag(ctx)
	default:
		return fmt.Errorf("unknown message type %q", msg.Type)
	}
}

// handleMove validates a client move against the current position, persists
// it, and publishes the resulting events.
func (c *gameConn) handleMove(ctx context.Context, req model.SendMove) error {
	lastFen, lastPly := chess.StartingPosition().String(), 0
	last, err := c.moves.GetLastMove(ctx, c.game.Id)
	switch {
	case err == nil:
		lastFen, lastPly = last.Fen, last.Ply
	case !errors.Is(err, pgx.ErrNoRows):
		return fmt.Errorf("loading last move: %w", err)
	}

	// odd plies are white's
	ply := lastPly + 1
	toMove, opponentWins := c.game.WhitePlayer, chess.BlackWon
	if ply%2 == 0 {
		toMove, opponentWins = c.game.BlackPlayer, chess.WhiteWon
	}
	if c.player != toMove {
		return errors.New("not your turn")
	}

	fen, err := chess.FEN(lastFen)
	if err != nil {
		return fmt.Errorf("bad stored position: %w", err)
	}
	board := chess.NewGame(fen)
	move, err := chess.UCINotation{}.Decode(board.Position(), req.UCI)
	if err != nil {
		return fmt.Errorf("decoding move: %w", err)
	}
	if err := board.Move(move); err != nil {
		return fmt.Errorf("illegal move: %w", err)
	}

	now := time.Now()
	whiteMS, blackMS := c.game.ClockInitial, c.game.ClockInitial
	if ply > 1 {
		whiteMS, blackMS = last.WhiteMS, last.BlackMS
	}
	if ply > 2 {
		mover := &whiteMS
		if ply%2 == 0 {
			mover = &blackMS
		}
		*mover -= int(now.Sub(last.CreatedAt).Milliseconds())
		if *mover <= 0 {
			if err := c.endGame(ctx, opponentWins.String(), "Timeout"); err != nil {
				slog.Error("flagging game", "game", c.game.Id, "err", err)
			}
			return errors.New("time expired")
		}
		*mover += c.game.ClockIncrement
	}

	created, err := c.moves.Create(ctx, model.CreateMove{
		GameID:    c.game.Id,
		Ply:       ply,
		UCI:       req.UCI,
		Fen:       board.Position().String(),
		WhiteMS:   whiteMS,
		BlackMS:   blackMS,
		CreatedAt: now,
	})
	if err != nil {
		return fmt.Errorf("saving move: %w", err)
	}

	slog.Info("got move", "move", created)
	if err := c.ps.Pub(ctx, events.NewMoveEvent(created)); err != nil {
		return err
	}

	if outcome := board.Outcome(); outcome != chess.NoOutcome {
		return c.endGame(ctx, outcome.String(), board.Method().String())
	}
	return nil
}

// handleFlag lets a player claim a win when the side to move has run out of
// time. The claim is verified against the last move's clock snapshot, so a
// premature claim is just rejected.
func (c *gameConn) handleFlag(ctx context.Context) error {
	if c.player != c.game.WhitePlayer && c.player != c.game.BlackPlayer {
		return errors.New("not a player")
	}

	last, err := c.moves.GetLastMove(ctx, c.game.Id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return errors.New("clock not running")
	case err != nil:
		return fmt.Errorf("loading last move: %w", err)
	}
	// each side's first move is untimed
	if last.Ply < 2 {
		return errors.New("clock not running")
	}

	remaining, result := last.WhiteMS, chess.BlackWon // white to move
	if last.Ply%2 == 1 {
		remaining, result = last.BlackMS, chess.WhiteWon
	}
	if time.Since(last.CreatedAt) < time.Duration(remaining)*time.Millisecond {
		return errors.New("time not expired")
	}
	return c.endGame(ctx, result.String(), "Timeout")
}

// endGame records the result if the game is still undecided and, if this
// caller won the race to decide it, publishes the game.end event.
func (c *gameConn) endGame(ctx context.Context, result, reason string) error {
	set, err := c.games.SetResult(ctx, c.game.Id, result)
	if err != nil {
		return fmt.Errorf("setting result: %w", err)
	}
	c.ended.Store(true)
	if !set {
		return nil
	}
	return c.ps.Pub(ctx, events.NewEndEvent(c.game.Id, result, reason))
}
