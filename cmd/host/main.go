package main

import (
	"github.com/charmbracelet/log"
	"github.com/tsukinoko-kun/gametube/internal/webrtc"
	"github.com/tsukinoko-kun/gametube/static"
	"net/http"
)

func main() {
	http.HandleFunc("/signaling", webrtc.SignalingHandler)
	http.HandleFunc("/", static.IndexHandler)

	if err := http.ListenAndServe(":4321", nil); err != nil {
		log.Fatal("failed to start server", "err", err)
	}
}
