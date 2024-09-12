package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log"

	"github.com/tsukinoko-kun/gametube/env"
	"github.com/tsukinoko-kun/gametube/game"
	"github.com/tsukinoko-kun/gametube/public"
	"github.com/tsukinoko-kun/gametube/view"
)

var (
	addr string
)

func init() {
	if err := env.EnsureCommonEnv(); err != nil {
		log.Fatal("failed to ensure common env variables", "err", err)
	}

	debug := flag.Bool("debug", false, "Enable debug logging")
	port := flag.Uint("port", 0, "Port to listen to")
	flag.Parse()

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	addr = fmt.Sprintf(":%d", *port)
}

func main() {
	killSig := make(chan os.Signal, 1)
	signal.Notify(killSig, os.Interrupt, syscall.SIGTERM)

	mux := http.NewServeMux()
	mux.HandleFunc("/", view.IndexHandler)
	mux.HandleFunc("/start/{slug}", game.StartHandler)
	mux.HandleFunc("/play", view.PlayHandler)
	mux.Handle("/public/", public.Handler)

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
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
}
