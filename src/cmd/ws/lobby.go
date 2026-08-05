package main

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/williamdann/chess-arena/internal/events"
	"github.com/williamdann/chess-arena/internal/pubsub"
)

func wsLobby(ps *pubsub.PubSub) wsHandler {
	return func(conn *websocket.Conn, _ *http.Request) {
		lobby, dropLobby := ps.Sub(events.TopicLobby)
		defer dropLobby()
		pres, dropPres := ps.Sub(events.TopicPresence)
		defer dropPres()

		go func() {
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					dropLobby()
					dropPres()
					return
				}
			}
		}()

		// receiving from a closed channel nils it out, disabling that case;
		// the loop ends once both subscriptions are gone
		for lobby != nil || pres != nil {
			var msg pubsub.Message
			var ok bool
			select {
			case msg, ok = <-lobby:
				if !ok {
					lobby = nil
					continue
				}
			case msg, ok = <-pres:
				if !ok {
					pres = nil
					continue
				}
			}
			if err := conn.WriteMessage(websocket.TextMessage, msg.Payload); err != nil {
				return
			}
		}
	}
}
