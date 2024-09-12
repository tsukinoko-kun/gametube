package view

import (
	"github.com/tsukinoko-kun/gametube/session"
	"net/http"
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

	_ = play("", s).Render(r.Context(), w)
}
