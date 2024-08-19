package webrtc

import (
	"encoding/json"
	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
	"net/http"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func SignalingHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("failed to upgrade connection", "err", err)
		return
	}
	defer func(conn *websocket.Conn) {
		err := conn.Close()
		log.Debug("WebSocket connection closed")
		if err != nil {
			log.Error("failed to close connection", "err", err)
		}
	}(conn)

	log.Debug("WebSocket connection established")

	wg, err := InitializePeerConnection(func(c AnswerCandidate) {
		response, err := json.Marshal(c)
		if err != nil {
			log.Error("failed to marshal ICE candidate", "err", err)
			return
		}

		if err := conn.WriteMessage(websocket.TextMessage, response); err != nil {
			log.Error("failed to send ICE candidate", "err", err)
		}
	})
	if err != nil {
		log.Error("failed to initialize peer connection", "err", err)
		return
	}
	defer wg.Done()

	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Error("failed to read message", "err", err)
			return
		}

		t, offer, candidate, err := Parse(p)
		if err != nil {
			log.Error("failed to parse message", "err", err)
			return
		}

		switch t {
		case MessageTypeOffer:
			log.Debug("received offer")
			answer, err := HandleOffer(offer)
			if err != nil {
				log.Error("failed to handle offer", "err", err)
				continue
			}

			response, err := json.Marshal(map[string]interface{}{
				"__message_type__": MessageTypeOffer,
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

		case MessageTypeCandidate:
			log.Debug("received ICE candidate")
			if err := HandleCandidate(candidate); err != nil {
				log.Error("failed to handle ICE candidate", "err", err)
				return
			}

		default:
			log.Error("unknown message type", "type", t)
			return
		}
	}
}
