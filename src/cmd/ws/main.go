package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/williamdann/chess-arena/internal/events"
	"github.com/williamdann/chess-arena/internal/postgres"
	"github.com/williamdann/chess-arena/internal/pubsub"
	"github.com/williamdann/chess-arena/internal/services"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return r.Header.Get("Origin") == "http://localhost:8080"
	},
}

type wsHandler func(*websocket.Conn, *http.Request)

func wsEcho(conn *websocket.Conn, _ *http.Request) {
	for {
		typ, data, err := conn.ReadMessage()
		if err != nil {
			slog.Error("ws error", "err", err)
			return
		}

		conn.WriteMessage(typ, data)
	}
}

func wsLobby(pubsub *pubsub.PubSub) wsHandler {
	return func(conn *websocket.Conn, _ *http.Request) {
		ch, drop := pubsub.Sub(events.TopicLobby)
		defer drop()

		go func() {
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					drop()
					return
				}
			}
		}()

		for msg := range ch {
			if err := conn.WriteMessage(websocket.TextMessage, msg.Payload); err != nil {
				return
			}
		}
	}
}

func wsGame(ps *pubsub.PubSub, games *postgres.GameStore, moves *postgres.MoveStore) wsHandler {
	return func(conn *websocket.Conn, r *http.Request) {
		id, err := uuid.Parse(r.URL.Query().Get("id"))
		if err != nil {
			conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "invalid game id"))
			return
		}

		// subscribe before snapshotting so no event lands between the two;
		// the client dedups snapshot/delta overlap by ply
		ch, drop := ps.Sub(events.GameTopic(id))
		defer drop()

		go func() {
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					drop()
					return
				}
			}
		}()

		game, err := games.GetById(r.Context(), id)
		if err != nil {
			conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "unknown game"))
			return
		}
		history, err := moves.GetForGame(r.Context(), id)
		if err != nil {
			slog.Error("failed to load moves", "game", id, "err", err)
			return
		}

		snapshot, err := json.Marshal(events.NewGameStateEvent(game, history))
		if err != nil {
			slog.Error("failed to marshal game snapshot", "game", id, "err", err)
			return
		}
		if err := conn.WriteMessage(websocket.TextMessage, snapshot); err != nil {
			return
		}

		for msg := range ch {
			if err := conn.WriteMessage(websocket.TextMessage, msg.Payload); err != nil {
				return
			}
		}
	}
}

func logTopic(ps *pubsub.PubSub, topic string) {
	ch, _ := ps.Sub(topic)
	go func() {
		for msg := range ch {
			slog.Info("pubsub event", "topic", msg.Topic, "payload", string(msg.Payload))
		}
	}()
}

func connectThen(handler wsHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// validate session
		_, ok := services.SessionFromContext(r.Context())
		if !ok {
			http.Error(w, "invalid session", http.StatusUnauthorized)
			return
		}

		// upgrade to a websocket connection
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("failed ws upgrade", "err", err)
			return
		}

		defer conn.Close()

		// call websocket handler
		handler(conn, r)
	}
}

func main() {
	// connect to database
	ctx := context.Background()
	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		slog.Error("invalid database config", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("could not reach database", "err", err)
		os.Exit(1)
	}

	users := postgres.NewUserStore(pool)
	sessions := postgres.NewSessionStore(pool)
	games := postgres.NewGameStore(pool)
	moves := postgres.NewMoveStore(pool)

	sessionService := services.NewSessionService(users, sessions)

	// start events listner
	pubsub := pubsub.NewPubSub(pool)
	go pubsub.Listen(ctx)

	// debug: log all lobby events
	logTopic(pubsub, events.TopicLobby)

	// start websocket server
	mux := http.NewServeMux()
	mux.Handle("GET /connect", sessionService.RequireAuth(connectThen(wsEcho)))

	mux.Handle("GET /lobby", sessionService.RequireAuth(connectThen(wsLobby(pubsub))))
	mux.Handle("GET /game", sessionService.RequireAuth(connectThen(wsGame(pubsub, games, moves))))

	slog.Info("started websocket server on 0.0.0.0:8081")
	http.ListenAndServe("0.0.0.0:8081", mux)
}
