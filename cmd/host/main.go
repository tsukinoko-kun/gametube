package main

import (
	"context"
	"errors"
	"github.com/charmbracelet/log"
	"github.com/tsukinoko-kun/gametube/internal/game"
	"github.com/tsukinoko-kun/gametube/internal/webrtc"
	"github.com/tsukinoko-kun/gametube/static"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

func main() {
	var g *game.Game

	for _, arg := range os.Args[1:] {
		switch arg {
		case "--debug":
			log.SetLevel(log.DebugLevel)
		default:
			if g != nil {
				log.Fatal("game binary already set", "new", arg, "old", g.Binary())
			}
			if _, err := os.Stat(arg); os.IsNotExist(err) {
				log.Fatal("game binary not found", "path", arg)
			}
			g = game.New(arg, filepath.Dir(arg))
			if err := g.Start(); err != nil {
				log.Fatal("failed to start game", "err", err)
			}
		}
	}

	killSig := make(chan os.Signal, 1)
	signal.Notify(killSig, os.Interrupt, os.Kill)

	http.HandleFunc("/signaling", webrtc.SignalingHandler)
	http.HandleFunc("/", static.IndexHandler)

	srv := &http.Server{
		Addr:    ":80",
		Handler: http.DefaultServeMux,
	}

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		log.Fatal("failed to listen", "err", err)
	}

	defer func(ln net.Listener) {
		if err := ln.Close(); err != nil {
			log.Error("failed to close listener", "err", err)
		}
	}(ln)

	log.Info("listening", "addr", ln.Addr().String())

	go func() {
		err := srv.Serve(ln)
		if err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				log.Info("server shutdown complete")
			} else {
				log.Error("server error", "err", err)
			}
		}
	}()

	<-killSig
	log.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server shutdown failed", "err", err)
	}

	if g != nil {
		if err := g.Stop(); err != nil {
			log.Error("failed to stop game", "err", err)
		}
	}
}
