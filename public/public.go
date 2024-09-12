package public

import (
	"embed"
	"net/http"
)

var (
	//go:embed *
	FS embed.FS

	Handler = http.StripPrefix("/public", http.FileServerFS(FS))
)
