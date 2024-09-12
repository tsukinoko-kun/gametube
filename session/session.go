package session

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

var (
	NoSessionError = errors.New("no session")
)

func generateSessionId() string {
	t := time.Now().Unix()
	r := rand.Int()
	return fmt.Sprintf("%d-%d", t, r)
}

func GetSessionId(w http.ResponseWriter, r *http.Request, create bool) (string, error) {
	// check if there is a session cookie
	sessionCookie, err := r.Cookie("session")
	if err != nil {
		if create {
			// there is no session cookie, create one
			sessionCookie = &http.Cookie{
				Name:  "session",
				Value: generateSessionId(),
			}
			http.SetCookie(w, sessionCookie)
		} else {
			return "", NoSessionError
		}
	}
	return sessionCookie.Value, nil
}
