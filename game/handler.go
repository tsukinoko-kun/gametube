package game

import (
	"errors"
	"net/http"

	"github.com/tsukinoko-kun/gametube/config"
	"github.com/tsukinoko-kun/gametube/session"
)

var (
	ErrInvalidGameSlug = errors.New("invalid game slug")
)

func getSlug(r *http.Request) (string, error) {
	s := r.PathValue("slug")
	_, found := config.FindGame(s)
	if found {
		return s, nil
	} else {
		return s, ErrInvalidGameSlug
	}
}

func StartHandler(w http.ResponseWriter, r *http.Request) {
	slug, err := getSlug(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	sessionId, err := session.GetSessionId(w, r, true)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	if g, ok := activeGames[sessionId]; ok && g != nil {
		go g.Stop()
		activeGames[sessionId] = nil
	}

	g, err := newGame(r.Context(), sessionId, slug)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	activeGames[sessionId] = g

	http.Redirect(w, r, "/play", http.StatusFound)
}
