package webrtc

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/pion/webrtc/v4"
)

var (
	peerConnection *webrtc.PeerConnection
)

type MessageType uint8

const (
	MessageTypeUnknown MessageType = iota
	MessageTypeOffer
	MessageTypeCandidate
)

type Message struct {
	MessageType MessageType `json:"__message_type__"`
}

type AnswerCandidate webrtc.ICECandidateInit

func Parse(data []byte) (messageType MessageType, offer *webrtc.SessionDescription, candidate *webrtc.ICECandidateInit, err error) {
	dataStr := string(data)
	_ = dataStr

	offer = &webrtc.SessionDescription{}
	candidate = &webrtc.ICECandidateInit{}

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return MessageTypeUnknown, nil, nil, errors.Join(errors.New("failed to unmarshal message"), err)
	}

	switch msg.MessageType {
	case MessageTypeOffer:
		if err := json.Unmarshal(data, &offer); err != nil {
			return MessageTypeOffer, nil, nil, errors.Join(errors.New("failed to unmarshal offer message"), err)
		}
		return MessageTypeOffer, offer, nil, nil

	case MessageTypeCandidate:
		if err := json.Unmarshal(data, candidate); err != nil {
			return MessageTypeCandidate, nil, nil, errors.Join(errors.New("failed to unmarshal candidate message"), err)
		}
		return MessageTypeCandidate, nil, candidate, nil
	default:
		return MessageTypeUnknown, nil, nil, fmt.Errorf("unknown message type: %q", msg.MessageType)
	}
}

func InitializePeerConnection(onICECandidate func(candidate AnswerCandidate)) error {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{
					"stun:stun1.l.google.com:19302",
					"stun:stun2.l.google.com:19302",
				},
			},
		},
	}

	var err error
	peerConnection, err = webrtc.NewPeerConnection(config)
	if err != nil {
		return errors.Join(errors.New("failed to create peer connection"), err)
	}

	peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		log.Info("new ICE candidate")
		onICECandidate(AnswerCandidate(c.ToJSON()))
	})

	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		d.OnOpen(func() {
			log.Info("data channel opened", "channel", d.Label())
		})
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			log.Info("received message", "channel", d.Label(), "message", string(msg.Data))
		})
	})

	if d, err := peerConnection.CreateDataChannel("foo", nil); err != nil {
		return errors.Join(errors.New("failed to create data channel"), err)
	} else {
		d.OnOpen(func() {
			if err := d.SendText("hello from foo"); err != nil {
				log.Error("failed to send message", "err", err)
			}
		})
	}
	return nil
}

func HandleOffer(offer *webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	if peerConnection == nil {
		return nil, errors.New("peer connection not initialized")
	}

	err := peerConnection.SetRemoteDescription(*offer)
	if err != nil {
		return nil, errors.Join(errors.New("failed to set remote description"), err)
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		return nil, errors.Join(errors.New("failed to create answer"), err)
	}

	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		return nil, errors.Join(errors.New("failed to set local description"), err)
	}

	return &answer, nil
}

func HandleCandidate(candidate *webrtc.ICECandidateInit) error {
	if peerConnection == nil {
		return errors.New("peer connection not initialized")
	}

	err := peerConnection.AddICECandidate(*candidate)
	if err != nil {
		return errors.Join(errors.New("failed to add ICE candidate"), err)
	}

	return nil
}
