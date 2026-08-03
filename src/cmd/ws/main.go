package main

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	// The handshake is a cross-origin HTTP request from the browser,
	// so you must decide which origins to allow.
	CheckOrigin: func(r *http.Request) bool {
		return r.Header.Get("Origin") == "http://localhost:8080"
	},
}

func handleConnect(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("failed ws upgrade", "err", err)
		return
	}

	defer conn.Close()

	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		// echo for test
		conn.WriteMessage(msgType, data)
	}
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /connect", handleConnect)

	slog.Info("started websocket server on 0.0.0.0:8081")
	http.ListenAndServe("0.0.0.0:8081", mux)
}
