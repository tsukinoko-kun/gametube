package view

import (
	"net/http"

	"github.com/tsukinoko-kun/gametube/game"
	"github.com/tsukinoko-kun/gametube/session"
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = session.GetSessionId(w, r, true)
	_ = index().Render(r.Context(), w)
}

func PlayHandler(w http.ResponseWriter, r *http.Request) {
	s, err := session.GetSessionId(w, r, false)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	g, ok := game.GetGame(s)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("no running game"))
		return
	}

	_ = play(g, s).Render(r.Context(), w)
}
