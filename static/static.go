package static

import (
	"embed"
	"net/http"
)

var (
	//go:embed *
	FS embed.FS
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, FS, "index.html")
}
