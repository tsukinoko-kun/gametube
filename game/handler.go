package game

import (
	"errors"
	"github.com/tsukinoko-kun/gametube/config"
	"github.com/tsukinoko-kun/gametube/session"
	"net/http"
)

var (
	ErrInvalidGameSlug = errors.New("invalid game slug")
)

func getSlug(r *http.Request) (string, error) {
	s := r.PathValue("slug")
	found := false
	for _, g := range config.Data.Games {
		if g.Slug == s {
			found = true
			break
		}
	}
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

	g, err := newGame(slug)
	activeGames[sessionId] = g

	http.Redirect(w, r, "/play", http.StatusFound)
}
