package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

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

	// start websocket server
	mux := http.NewServeMux()
	mux.Handle("GET /lobby", sessionService.RequireAuth(connectThen(wsLobby(pubsub))))
	mux.Handle("GET /game/{id}", sessionService.RequireAuth(connectThen(wsGame(games, moves, pubsub))))

	slog.Info("started websocket server on 0.0.0.0:8081")
	http.ListenAndServe("0.0.0.0:8081", mux)
}
