package handler

import (
	"net/http"

	"app/internal/view/page"
)

func HomePage(w http.ResponseWriter, r *http.Request) error {
	return Render(w, r, page.Home())
}
