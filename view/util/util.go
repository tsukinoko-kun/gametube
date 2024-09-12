package util

import (
	"github.com/a-h/templ"
	"path"
)

func Join(elem ...string) templ.SafeURL {
	return templ.SafeURL(path.Join(elem...))
}
