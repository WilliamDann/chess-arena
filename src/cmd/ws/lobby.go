package main

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/williamdann/chess-arena/internal/events"
	"github.com/williamdann/chess-arena/internal/pubsub"
)

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
