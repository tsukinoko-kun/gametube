package main

import (
	"encoding/json"
	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
	"github.com/tsukinoko-kun/gametube/internal/webrtc"
	"github.com/tsukinoko-kun/gametube/static"
	"net/http"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("failed to upgrade connection", "err", err)
		return
	}
	defer func(conn *websocket.Conn) {
		err := conn.Close()
		log.Info("WebSocket connection closed")
		if err != nil {
			log.Error("failed to close connection", "err", err)
		}
	}(conn)

	log.Info("WebSocket connection established")

	if err := webrtc.InitializePeerConnection(func(c webrtc.AnswerCandidate) {
		response, err := json.Marshal(c)
		if err != nil {
			log.Error("failed to marshal ICE candidate", "err", err)
			return
		}

		if err := conn.WriteMessage(websocket.TextMessage, response); err != nil {
			log.Error("failed to send ICE candidate", "err", err)
		}
	}); err != nil {
		log.Error("failed to initialize peer connection", "err", err)
		return
	}

	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Error("failed to read message", "err", err)
			return
		}

		t, offer, candidate, err := webrtc.Parse(p)
		if err != nil {
			log.Error("failed to parse message", "err", err)
			return
		}

		switch t {
		case webrtc.MessageTypeOffer:
			log.Info("received offer")
			answer, err := webrtc.HandleOffer(offer)
			if err != nil {
				log.Error("failed to handle offer", "err", err)
				continue
			}

			response, err := json.Marshal(map[string]interface{}{
				"__message_type__": webrtc.MessageTypeOffer,
				"sdp":              answer.SDP,
				"type":             answer.Type.String(),
			})
			if err != nil {
				log.Error("failed to marshal answer", "err", err)
				continue
			}

			if err := conn.WriteMessage(websocket.TextMessage, response); err != nil {
				log.Error("failed to send answer", "err", err)
				return
			}

		case webrtc.MessageTypeCandidate:
			log.Info("received ICE candidate")
			if err := webrtc.HandleCandidate(candidate); err != nil {
				log.Error("failed to handle ICE candidate", "err", err)
				return
			}

		default:
			log.Error("unknown message type", "type", t)
			return
		}
	}
}

func main() {
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/", static.IndexHandler)

	if err := http.ListenAndServe(":4321", nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
